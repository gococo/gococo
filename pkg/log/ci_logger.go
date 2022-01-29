package log

import "go.uber.org/zap"

type ciLogger struct {
	logger       *zap.Logger
	exitCallback func()
}

func (l *ciLogger) Debugf(format string, args ...interface{}) {
	l.logger.Sugar().Debugf(format, args...)
}

func (l *ciLogger) Donef(format string, args ...interface{}) {
	l.logger.Sugar().Infof(format, args...)
}

func (l *ciLogger) Infof(format string, args ...interface{}) {
	l.logger.Sugar().Infof(format, args...)
}

func (l *ciLogger) Warnf(format string, args ...interface{}) {
	l.logger.Sugar().Warnf(format, args...)
}

func (l *ciLogger) Fatalf(format string, args ...interface{}) {
	if l.exitCallback != nil {
		l.exitCallback()
	}

	l.logger.Sugar().Fatalf(format, args...)
}

func (l *ciLogger) Errorf(format string, args ...interface{}) {
	l.logger.Sugar().Errorf(format, args...)
}

func (l *ciLogger) StartWait(message string) {
}

func (l *ciLogger) StopWait() {
}

func (l *ciLogger) Sync() {
	l.logger.Sync()
}

func (l *ciLogger) SetExitCallback(callback func()) {
	l.exitCallback = callback
}

func newCiLogger() *ciLogger {
	logger, _ := zap.NewDevelopment()
	logger = logger.WithOptions(zap.AddCallerSkip(2))

	return &ciLogger{
		logger: logger,
	}
}
