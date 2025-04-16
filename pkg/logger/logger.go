package logger

import (
	"github.com/google/wire"
)

// ProviderSet is log providers.
var ProviderSet = wire.NewSet(NewZapLogger, wire.Bind(new(Logger), new(*ZapLogger)))

// Logger is logger interface.
// 定义通用接口，可以使用不同的日志库实现
type Logger interface {
	Debug(msg string, fields ...Field)
	Debugf(format string, args ...interface{})
	Info(msg string, fields ...Field)
	Infof(format string, args ...interface{})
	Warn(msg string, fields ...Field)
	Warnf(format string, args ...interface{})
	Error(msg string, fields ...Field)
	Errorf(format string, args ...interface{})
	Panic(msg string, fields ...Field)
	Panicf(format string, args ...interface{})
	Fatal(msg string, fields ...Field)
	Fatalf(format string, args ...interface{})
	SetLevel(level Level)
	Sync() error
}
