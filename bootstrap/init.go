package bootstrap

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jinzhu/configor"
	"github.com/jinzhu/now"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/real-web-world/hh-lol-prophet/services/db/models"

	"github.com/real-web-world/hh-lol-prophet/pkg/windows/admin"
	"github.com/real-web-world/hh-lol-prophet/services/ws"

	hh_lol_prophet "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/services/buffApi"

	"github.com/real-web-world/hh-lol-prophet/conf"
	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
	"github.com/real-web-world/hh-lol-prophet/pkg/logger"
)

const (
	defaultTZ = "Asia/Shanghai"
)

func initConf() {
	_ = godotenv.Load(".env")
	if bdk.IsFile(".env.local") {
		_ = godotenv.Overload(".env.local")
	}
	// confPath := "./config/config.json"
	// err := configor.Load(global.Conf, confPath)
	*global.Conf = global.DefaultAppConf
	err := configor.Load(global.Conf)
	if err != nil {
		panic(err)
	}
	err = initClientConf()
	if err != nil {
		panic(err)
	}
}

func initClientConf() (err error) {
	dbPath := conf.SqliteDBPath
	var db *gorm.DB
	var dbLogger = gormLogger.Discard
	if !global.IsDevMode() {
		dbLogger = gormLogger.Default
	}
	gormCfg := &gorm.Config{
		Logger: dbLogger,
	}
	if !bdk.IsFile(dbPath) {
		db, err = gorm.Open(sqlite.Open(dbPath), gormCfg)
		bts, _ := json.Marshal(global.DefaultClientConf)
		err = db.Exec(models.InitLocalClientSql, models.LocalClientConfKey, string(bts)).Error
		if err != nil {
			return
		}
		*global.ClientConf = global.DefaultClientConf
	} else {
		db, err = gorm.Open(sqlite.Open(dbPath), gormCfg)
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

func initLog(cfg *conf.LogConf) {
	writeSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filepath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	})
	if global.IsDevMode() {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeDuration = zapcore.StringDurationEncoder
	level, err := logger.Str2ZapLevel(cfg.Level)
	if err != nil {
		panic("zap level is Incorrect")
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		writeSyncer,
		zap.NewAtomicLevelAt(level),
	)
	global.Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
}
func InitApp() error {
	admin.MustRunWithAdmin()
	initConf()
	initLog(&global.Conf.Log)
	initLib()
	initApi()
	initGlobal()
	return nil
}

func initGlobal() {
	go initAutoReloadCalcConf()
}

func initAutoReloadCalcConf() {
	ticker := time.NewTicker(time.Minute)
	for {
		latestScoreConf, err := buffApi.GetCurrConf()
		if err == nil && latestScoreConf != nil && latestScoreConf.Enabled {
			global.SetScoreConf(*latestScoreConf)
		}
		<-ticker.C
	}
}

func initApi() {
	buffApi.Init(global.Conf.BuffApi.Url, global.Conf.BuffApi.Timeout)
}
func initLib() {
	_ = os.Setenv("TZ", defaultTZ)
	now.WeekStartDay = time.Monday
	go func() {
		initUserInfo()
		if global.Conf.Sentry.Enabled {
			_ = initSentry(global.Conf.Sentry.Dsn)
		}
	}()
	ws.Init()
}

func initUserInfo() {
	resp, err := http.Get("https://api.ip.sb/ip")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	bts, _ := io.ReadAll(resp.Body)
	global.SetUserInfo(global.UserInfo{
		IP: strings.Trim(string(bts), "\n"),
		// Mac:   windows.GetMac(),
		// CpuID: windows.GetCpuID(),
	})
}
func initSentry(dsn string) error {
	isDebugMode := global.IsDevMode()
	sampleRate := 1.0
	if !isDebugMode {
		sampleRate = 1.0
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Debug:       isDebugMode,
		SampleRate:  sampleRate,
		Release:     hh_lol_prophet.Commit,
		Environment: global.GetEnv(),
	})
	if err == nil {
		global.Cleanups["sentryFlush"] = func() error {
			sentry.Flush(2 * time.Second)
			return nil
		}
		userInfo := global.GetUserInfo()
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetContext("buffgeDefault", map[string]interface{}{
				"ip": userInfo.IP,
				// "mac":   userInfo.Mac,
				// "cpuID": userInfo.CpuID,
			})
			scope.SetUser(sentry.User{
				// ID:        userInfo.Mac,
				IPAddress: userInfo.IP,
			})
			// scope.SetExtra("cpuID", userInfo.CpuID)
		})
	}
	return err
}
