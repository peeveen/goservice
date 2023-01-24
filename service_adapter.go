package goservice

import (
	"io"

	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
)

type serviceAdapter struct {
	serviceRunner ServiceRunner
	loggingConfig *LoggingConfig
	logger        *logrus.Logger
}

type LoggingConfig struct {
	LogFilename      string
	ErrorLogFilename string
	Debug            bool
}

func newServiceAdapter(serviceRunner ServiceRunner, loggingConfig *LoggingConfig, logger *logrus.Logger) *serviceAdapter {
	return &serviceAdapter{serviceRunner, loggingConfig, logger}
}

// Called in service mode, when requested to start.
func (a *serviceAdapter) Start(s service.Service) error {
	// Create a system logger. If this fails, it's very bad news.
	svcLogger, err := s.Logger(nil)
	if err != nil {
		a.logger.Fatal(err)
		return err
	}
	// We can stop using console logging now. The service logger,
	// if running interactively, will log to console anyway. If
	// not running interactively, logging will go to service logs.
	a.logger.Out = io.Discard

	a.addLoggerHooks(svcLogger)

	if service.Interactive() {
		handleCtrlC(func() { a.serviceRunner.Stop() })
		a.serviceRunner.Run()
	} else {
		a.serviceRunner.Start()
	}

	return err
}

// Called in service mode, when requested to stop.
// This function should ideally return within a few seconds.
func (a *serviceAdapter) Stop(s service.Service) error {
	return a.serviceRunner.Stop()
}

func (a *serviceAdapter) addLoggerHooks(svcLogger service.Logger) {
	// First hook is the service logger
	a.logger.AddHook(NewServiceLoggerHook(svcLogger, a.loggingConfig != nil && a.loggingConfig.Debug))

	if a.loggingConfig != nil {
		// Then two hooks for output files.
		if a.loggingConfig.LogFilename != "" {
			runLogLevel := logrus.InfoLevel
			if a.loggingConfig.Debug {
				runLogLevel = logrus.DebugLevel
			}
			runLogHook, err := NewFileLoggerHook(a.loggingConfig.LogFilename, true, runLogLevel, 50, 28)
			if err != nil {
				a.logger.Error(err.Error())
			} else {
				a.logger.AddHook(runLogHook)
			}
		}

		if a.loggingConfig.ErrorLogFilename != "" {
			runLogHook, err := NewFileLoggerHook(a.loggingConfig.ErrorLogFilename, true, logrus.ErrorLevel, 50, 28)
			if err != nil {
				a.logger.Error(err.Error())
			} else {
				a.logger.AddHook(runLogHook)
			}
		}
	}
}
