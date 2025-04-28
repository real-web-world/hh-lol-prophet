package bootstrap

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jinzhu/configor"
	"github.com/jinzhu/now"
	"github.com/joho/godotenv"
	"github.com/real-web-world/bdk"
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

	hhLolProphet "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/conf"
	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/pkg/os/admin"
	"github.com/real-web-world/hh-lol-prophet/services/buffApi"
	"github.com/real-web-world/hh-lol-prophet/services/db/models"
)

const (
	DefaultTZ         = "Asia/Shanghai"
	EnvFileName       = ".env"
	EnvLocalFileName  = ".env.local"
	LocalConfFilePath = "./config.json"
)

func getRemoteConf() (*conf.AppConf, error) {
	cli := http.Client{
		Timeout: time.Second * 2,
	}
	resp, err := cli.Get(conf.GetRemoteConfApi)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	bts, _ := io.ReadAll(resp.Body)
	type BuffResp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	res := &BuffResp{}
	if err = json.Unmarshal(bts, res); err != nil {
		return nil, err
	}
	cfg := &conf.AppConf{}
	if err = json.Unmarshal(res.Data, cfg); err != nil {
		return nil, err
	}
	if cfg.AppName == "" {
		return nil, errors.New("获取远程配置失败")
	}
	return cfg, nil
}
func initConf() {
	_ = godotenv.Load(EnvFileName)
	if bdk.IsFile(EnvLocalFileName) {
		_ = godotenv.Overload(EnvLocalFileName)
	}
	remoteConfC := make(chan *conf.AppConf, 1)
	go func() {
		defer func() {
			_ = recover()
		}()
		if global.IsEnvModeDev() {
			remoteConfC <- nil
			return
		}
		cfg, _ := getRemoteConf()
		if cfg != nil {
			bts, _ := json.Marshal(cfg)
			_ = os.WriteFile(LocalConfFilePath, bts, 0664)
		}
		remoteConfC <- cfg
	}()
	*global.Conf = global.DefaultAppConf
	if err := initClientConf(); err != nil {
		log.Fatalf("本地配置错误,请删除%s文件后重启,错误信息:%v", conf.SqliteDBPath, err)
	}
	remoteConf := <-remoteConfC
	if remoteConf == nil {
		localConfFiles := make([]string, 0, 1)
		if bdk.IsFile(LocalConfFilePath) {
			localConfFiles = append(localConfFiles, LocalConfFilePath)
		}
		if err := configor.Load(global.Conf, localConfFiles...); err != nil {
			log.Fatalf("本地配置错误:%v", err)
		}
	} else {
		global.Conf = remoteConf
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
		bts, _ := json.Marshal(global.DefaultClientUserConf)
		err = db.Exec(models.InitLocalClientSql, models.LocalClientConfKey, string(bts)).Error
		if err != nil {
			return
		}
		*global.ClientUserConf = global.DefaultClientUserConf
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
		localClientConf := &conf.ClientUserConf{}
		err = json.Unmarshal([]byte(confItem.Val), localClientConf)
		if err != nil || conf.ValidClientUserConf(localClientConf) != nil {
			return errors.New("本地配置错误")
		}
		global.ClientUserConf = localClientConf
	}
	global.SqliteDB = db
	return nil
}

func initLog(appName string) {
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
	if global.IsProdMode() {
		bufWriter := bufio.NewWriter(log.Writer())
		logWriter := bdk.NewConcurrentWriter(bufWriter)
		log.SetOutput(logWriter)
		global.SetCleanup(global.LogWriterCleanupKey, logWriter.Close)
		core = otelzap.NewCore(appName,
			otelzap.WithLoggerProvider(otelLogGlobal.GetLoggerProvider()),
		)
	}
	global.Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2)).Sugar()
	if global.IsProdMode() {
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
	cfg := global.Conf
	if err := initOtel(context.Background(), cfg.Mode, cfg.AppName, cfg.Log, cfg.Otlp,
		global.GetUserInfo()); err != nil {
		return err
	}
	initLog(cfg.AppName)
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
		initApi(cfg.BuffApi)
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
	// 废弃
	//go initAutoReloadCalcConf()
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

func initApi(buffApiCfg conf.BuffApi) {
	buffApi.Init(buffApiCfg.Url, buffApiCfg.Timeout)
}

func initLib() error {
	_ = os.Setenv("TZ", DefaultTZ)
	now.WeekStartDay = time.Monday
	return nil
}

func initUserInfo() {
	hBts := sha256.Sum256(binary.LittleEndian.AppendUint64(nil, bdk.GetMac()))
	global.SetUserMac(hex.EncodeToString(hBts[:]))
}

func initOtel(ctx context.Context, mode conf.Mode, appName string,
	logConf conf.LogConf, otlpCfg conf.OtlpConf, userInfo global.UserInfo) error {
	res, err := newResource(mode, appName, userInfo)
	if err != nil {
		return err
	}
	loggerProvider, err := newLoggerProvider(ctx, res, logConf, otlpCfg)
	if err != nil {
		return err
	}
	global.SetCleanup(global.OtelCleanupKey, func(c context.Context) error {
		return loggerProvider.Shutdown(c)
	})
	otelLogGlobal.SetLoggerProvider(loggerProvider)
	return nil
}
func newResource(mode conf.Mode, appName string, userInfo global.UserInfo) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(appName),
			semconv.ServiceVersion(hhLolProphet.APPVersion),
			attribute.String("buff.userMac", userInfo.MacHash),
			attribute.String("buff.commitID", hhLolProphet.Commit),
			attribute.String("buff.mode", mode),
		))
}

func newLoggerProvider(ctx context.Context, res *resource.Resource,
	logConf conf.LogConf, otlpCfg conf.OtlpConf) (*otelLog.LoggerProvider, error) {
	exporter, err := otlploghttp.New(ctx, otlploghttp.WithEndpointURL(otlpCfg.EndpointUrl+"/v1/logs"),
		otlploghttp.WithHeaders(map[string]string{
			"Authorization": "Basic " + otlpCfg.Token,
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
		otelLog.WithProcessor(minsev.NewLogProcessor(processor, conf.LogLevel2Otel(logConf.Level))),
	)
	return provider, nil
}
