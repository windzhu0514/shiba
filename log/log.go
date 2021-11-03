package log

import (
	"io"
	"net/http"
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	RotateModeSize  = "size"
	RotateModeDaily = "daily"

	EncoderModeConsole = "console"
	EncoderModeJson    = "json"
)

type Level int8

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	DPanicLevel
	PanicLevel
	FatalLevel

	_minLevel = DebugLevel
	_maxLevel = FatalLevel
)

type Config struct {
	EncoderMode   string `json:"encoderMode" yaml:"encoderMode"`
	RotatorMode   string `json:"rotatorMode" yaml:"rotatorMode"`
	Level         Level  `json:"level" yaml:"level"`
	WithoutCaller bool   `json:"withoutCaller" yaml:"withoutCaller"`
	FileName      string `json:"fileName" yaml:"fileName"`
	MaxSize       int    `json:"maxSize" yaml:"maxSize"`
	MaxAge        int    `json:"maxAge" yaml:"maxAge"`
	MaxBackups    int    `json:"maxBackups" yaml:"maxBackups"`
	UTCTime       bool   `json:"utcTime" yaml:"utcTime"`
	Compress      bool   `json:"compress" yaml:"compress"`
}

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	With(args ...interface{}) Logger
	SetLevel(l Level)
	Clone(name string) Logger
	Close() error
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

var defaultLogger = New("", nil, Config{})

func New(name string, w io.WriteCloser, cfg Config) Logger {
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:       "msg",
		LevelKey:         "level",
		TimeKey:          "ts",
		NameKey:          "logger",
		CallerKey:        "caller",
		FunctionKey:      zapcore.OmitKey,
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      zapcore.LowercaseLevelEncoder,
		EncodeTime:       zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.999"),
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		EncodeName:       zapcore.FullNameEncoder,
		ConsoleSeparator: " ",
	}

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	if cfg.EncoderMode == EncoderModeJson {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var ws []io.Writer
	if len(cfg.FileName) > 0 {
		var rotator io.Writer = &lumberjack.Logger{
			Filename:   cfg.FileName,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			LocalTime:  !cfg.UTCTime,
			Compress:   cfg.Compress,
		}

		if cfg.RotatorMode == RotateModeDaily {
			rotator = &dailyRotator{
				Filename:  cfg.FileName,
				MaxAge:    cfg.MaxAge,
				LocalTime: !cfg.UTCTime,
				Compress:  cfg.Compress,
			}
		}

		ws = append(ws, rotator)
	} else {
		ws = append(ws, os.Stdout)
	}

	if w != nil {
		ws = append(ws, w)
	}

	level := zap.NewAtomicLevelAt(zapcore.Level(cfg.Level - 1))
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(io.MultiWriter(ws...)),
		level,
	)

	opts := []zap.Option{zap.AddCallerSkip(1)}
	if !cfg.WithoutCaller {
		opts = append(opts, zap.AddCaller())
	}

	return &logger{
		logger: zap.New(core, opts...).Sugar(),
		level:  level,
	}
}

func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Info(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

func Warn(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Warnf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Error(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Panic(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Panicf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Fatal(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func With(args ...interface{}) Logger {
	return defaultLogger.With(args...)
}

func SetLevel(lvl Level) {
	defaultLogger.SetLevel(lvl)
}

func Clone(name string) Logger {
	return defaultLogger.Clone(name)
}

func Close() error {
	return defaultLogger.Close()
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defaultLogger.ServeHTTP(w, r)
}

type logger struct {
	logger *zap.SugaredLogger
	level  zap.AtomicLevel
}

func (l *logger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *logger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *logger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

func (l *logger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *logger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *logger) Panic(args ...interface{}) {
	l.logger.Panic(args...)
}

func (l *logger) Panicf(format string, args ...interface{}) {
	l.logger.Panicf(format, args...)
}

func (l *logger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *logger) With(args ...interface{}) Logger {
	return &logger{
		logger: l.logger.With(args...),
		level:  l.level,
	}
}

func (l *logger) SetLevel(lvl Level) {
	l.level.SetLevel(zapcore.Level(lvl - 1))
}

func (l *logger) Clone(name string) Logger {
	return &logger{
		logger: l.logger.Named(name),
		level:  l.level,
	}
}

func (l *logger) Close() error {
	return l.logger.Sync()
}

func (l *logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.level.ServeHTTP(w, r)
}
