package bootstrap

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/jinzhu/configor"
	"github.com/jinzhu/now"
	"github.com/joho/godotenv"
	"github.com/real-web-world/bdk"
	hhLolProphet "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/conf"
	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/pkg/os/admin"
	"github.com/real-web-world/hh-lol-prophet/services/buffApi"
	"github.com/real-web-world/hh-lol-prophet/services/db/models"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/processors/minsev"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otelLogGlobal "go.opentelemetry.io/otel/log/global"
	otelLog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const (
	defaultTZ        = "Asia/Shanghai"
	envFileName      = ".env"
	envLocalFileName = ".env.local"
)

func initConf() {
	_ = godotenv.Load(envFileName)
	if bdk.IsFile(envLocalFileName) {
		_ = godotenv.Overload(envLocalFileName)
	}
	err := initClientConf()
	if err != nil {
		panic(err)
	}

	*global.Conf = global.DefaultAppConf
	err = configor.Load(global.Conf)
	if err != nil {
		panic(err)
	}

}

func initClientConf() (err error) {
	dbPath := conf.SqliteDBPath
	var db *gorm.DB
	var dbLogger = gormLogger.Discard
	if global.IsDevMode() {
		dbLogger = gormLogger.Default
	}
	gormCfg := &gorm.Config{
		Logger: dbLogger,
	}
	if !bdk.IsFile(dbPath) {
		db, err = gorm.Open(sqlite.Open(dbPath), gormCfg)
		if err != nil {
			log.Fatalln("创建配置文件失败")
		}
		bts, _ := json.Marshal(global.DefaultClientConf)
		err = db.Exec(models.InitLocalClientSql, models.LocalClientConfKey, string(bts)).Error
		if err != nil {
			return
		}
		*global.ClientConf = global.DefaultClientConf
	} else {
		db, err = gorm.Open(sqlite.Open(dbPath), gormCfg)
		if err != nil {
			log.Fatalln("配置文件错误,请删除配置文件重试")
		}
		confItem := &models.Config{}
		err = db.Table("config").Where("k = ?", models.LocalClientConfKey).First(confItem).Error
		if err != nil {
			return
		}
		localClientConf := &conf.Client{}
		err = json.Unmarshal([]byte(confItem.Val), localClientConf)
		if err != nil || conf.ValidClientConf(localClientConf) != nil {
			return errors.New("本地配置错误")
		}
		global.ClientConf = localClientConf
	}
	global.SqliteDB = db
	return nil
}

func initLog() {
	cfg := global.Conf
	ws := zapcore.AddSync(log.Writer())
	logLevel := zapcore.DebugLevel
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeDuration = zapcore.StringDurationEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		ws,
		zap.NewAtomicLevelAt(logLevel),
	)
	if !global.IsDevMode() {
		bufWriter := bufio.NewWriter(log.Writer())
		logWriter := bdk.NewConcurrentWriter(bufWriter)
		log.SetOutput(logWriter)
		global.SetCleanup(global.LogWriterCleanupKey, logWriter.Close)
		core = otelzap.NewCore(cfg.ProjectUrl,
			otelzap.WithLoggerProvider(otelLogGlobal.GetLoggerProvider()),
		)
	}
	global.Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2)).Sugar()
	if !global.IsDevMode() {
		global.SetCleanup(global.ZapLoggerCleanupKey, func(_ context.Context) error {
			return global.Logger.Sync()
		})
	}
	return
}
func InitApp() error {
	admin.MustRunWithAdmin()
	initConf()
	initUserInfo()
	if err := initOtel(context.Background()); err != nil {
		return err
	}
	initLog()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		initConsole()
		return nil
	})
	g.Go(func() error {
		return initLib()
	})
	g.Go(func() error {
		initApi()
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}
	initGlobal()
	return nil
}

func initConsole() {
	initConsoleAdapt()
}

func initGlobal() {
	go initAutoReloadCalcConf()
}

func initAutoReloadCalcConf() {
	ticker := time.NewTicker(time.Minute)
	for {
		latestScoreConf, err := buffApi.GetClientConf()
		if err == nil {
			if latestScoreConf.Enabled {
				global.SetScoreConf(*latestScoreConf)
			}
		}
		<-ticker.C
	}
}

func initApi() {
	buffApi.Init(global.Conf.BuffApi.Url, global.Conf.BuffApi.Timeout)
}

func initLib() error {
	_ = os.Setenv("TZ", defaultTZ)
	now.WeekStartDay = time.Monday
	return nil
}

func initUserInfo() {
	sha1.New()
	hBts := sha1.Sum(binary.LittleEndian.AppendUint64(nil, bdk.GetMac()))
	global.SetUserMac(
		hex.EncodeToString(hBts[:]),
	)
}

func initOtel(ctx context.Context) error {
	res, err := newResource()
	if err != nil {
		return err
	}
	loggerProvider, err := newLoggerProvider(ctx, res)
	if err != nil {
		return err
	}
	global.SetCleanup(global.OtelCleanupKey, func(c context.Context) error {
		return loggerProvider.Shutdown(c)
	})
	otelLogGlobal.SetLoggerProvider(loggerProvider)
	return nil
}
func newResource() (*resource.Resource, error) {
	cfg := global.Conf
	userInfo := global.GetUserInfo()
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(cfg.AppName),
			semconv.ServiceVersion(hhLolProphet.APPVersion),
			attribute.String("buff.userMac", userInfo.MacHash),
			attribute.String("buff.commitID", hhLolProphet.Commit),
			attribute.String("buff.mode", global.Conf.Mode),
		))
}

func newLoggerProvider(ctx context.Context, res *resource.Resource) (*otelLog.LoggerProvider, error) {
	cfg := global.Conf.Otlp
	exporter, err := otlploghttp.New(ctx, otlploghttp.WithEndpointURL(cfg.EndpointUrl+"/v1/logs"),
		otlploghttp.WithHeaders(map[string]string{
			"Authorization": "Basic " + cfg.Token,
		}),
		otlploghttp.WithRetry(otlploghttp.RetryConfig{
			Enabled:         true,
			InitialInterval: time.Second,
			MaxInterval:     time.Second * 5,
			MaxElapsedTime:  30 * time.Minute,
		}),
	)
	if err != nil {
		return nil, err
	}
	processor := otelLog.NewBatchProcessor(exporter, otelLog.WithExportInterval(time.Second))
	provider := otelLog.NewLoggerProvider(
		otelLog.WithResource(res),
		otelLog.WithProcessor(minsev.NewLogProcessor(processor, minsev.SeverityInfo)),
	)
	return provider, nil
}
