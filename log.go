package log

import (
	"fmt"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	dateRollingSuffix = ".%Y%m%d"
	RollingBySize     = 0
	RollingByDate     = 1
)

var (
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
	once   sync.Once
)

func Init(filename string, logLevel string, maxSize, maxBackups, maxAge int, rollingBy int, stdout ...bool) {
	switch rollingBy {
	case RollingBySize:
		SizeRolling(filename, logLevel, maxSize, maxBackups, maxAge, stdout...)
	case RollingByDate:
		DateRolling(filename, logLevel, maxBackups, maxAge, stdout...)
	}
}

// log time coder
func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.RFC3339))
}

func logLv(logLevel string) zapcore.Level {
	level := zapcore.InfoLevel
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = zapcore.DebugLevel
	case "INFO", "": // make the zero value useful
		level = zapcore.InfoLevel
	case "WARN":
		level = zapcore.WarnLevel
	case "ERROR":
		level = zapcore.ErrorLevel
	case "DPANIC":
		level = zapcore.DPanicLevel
	case "PANIC":
		level = zapcore.PanicLevel
	case "FATAL":
		level = zapcore.FatalLevel
	default:
		fmt.Printf("invalid log level %s, change to INFO\n", logLevel)
		level = zapcore.InfoLevel
	}
	return level
}

func DateRolling(filename string, logLevel string, maxBackups, maxAge int, stdout ...bool) {
	rotateLogs, err := RotateLogs(filename, uint(maxBackups), maxAge)
	if err != nil {
		panic(err)
	}
	level := logLv(logLevel)
	cores := make([]zapcore.Core, 0)
	fileWriterSyncer := zapcore.AddSync(rotateLogs)
	logCore(fileWriterSyncer, level, &cores)
	devCore(stdout, level, &cores)
	core := zapcore.NewTee(cores...)
	//skip one layer of caller tracing
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Logger.Sugar()
}

func SizeRolling(filename string, logLevel string, maxSize, maxBackups, maxAge int, stdout ...bool) {
	level := logLv(logLevel)
	cores := make([]zapcore.Core, 0)
	fileWriterSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize, //MB
		LocalTime:  true,
		MaxBackups: maxBackups,
		MaxAge:     maxAge, //Day
		Compress:   true,   //compress log file
	})
	logCore(fileWriterSyncer, level, &cores)
	devCore(stdout, level, &cores)
	core := zapcore.NewTee(cores...)
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Logger.Sugar()
}

func logCore(fileWriterSyncer zapcore.WriteSyncer, level zapcore.Level, cores *[]zapcore.Core) {
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = timeEncoder
	fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	c := zapcore.NewCore(zapcore.NewConsoleEncoder(fileEncoderConfig), fileWriterSyncer, level)
	//format log output time & uppercase log level
	*cores = append(*cores, c)
}

func devCore(stdout []bool, level zapcore.Level, cores *[]zapcore.Core) {
	if len(stdout) > 0 && stdout[0] {
		developmentEncoderConfig := zap.NewDevelopmentEncoderConfig()
		developmentEncoderConfig.EncodeTime = timeEncoder
		developmentEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		c := zapcore.NewCore(zapcore.NewConsoleEncoder(developmentEncoderConfig), zapcore.WriteSyncer(os.Stdout), level)
		*cores = append(*cores, c)
	}
}

// RotateLogs rotate log file
func RotateLogs(filePath string, maxBackups uint, maxAge int) (*rotatelogs.RotateLogs, error) {
	//rotate log suffix：date info
	var filename string
	ext := filepath.Ext(filePath)
	if len(ext) > 0 {
		filename = strings.TrimSuffix(filePath, ext) + dateRollingSuffix + ext
	} else {
		filename = filePath + dateRollingSuffix
	}
	options := []rotatelogs.Option{rotatelogs.WithLinkName(filePath), rotatelogs.WithRotationTime(time.Hour * 24)}
	if int(maxBackups) > maxAge {
		options = append(options, rotatelogs.WithMaxAge(time.Hour*24*time.Duration(maxAge)))
	} else {
		options = append(options, rotatelogs.WithRotationCount(maxBackups))
	}
	return rotatelogs.New(filename, options...)
}

func Default() {
	once.Do(func() {
		developmentEncoderConfig := zap.NewDevelopmentEncoderConfig()
		developmentEncoderConfig.EncodeTime = timeEncoder
		developmentEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		//log default level: info
		logLevel := zap.InfoLevel
		//check the debug switch
		debugEnabled := os.Getenv("DEBUG")
		if len(debugEnabled) > 0 {
			logLevel = zap.DebugLevel
		}
		core := zapcore.NewCore(zapcore.NewConsoleEncoder(developmentEncoderConfig), zapcore.WriteSyncer(os.Stdout), logLevel)
		Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		Sugar = Logger.Sugar()
	})
}

func Info(args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Info(args...)
}

func Infof(template string, args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Infof(template, args...)
}

func Debug(args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Debugf(template, args...)
}

func Warn(args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Warnf(template, args...)
}

func Error(args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Errorf(template, args...)
}

func Fatal(args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	if Sugar == nil {
		Default()
	}
	Sugar.Fatalf(template, args...)
}
