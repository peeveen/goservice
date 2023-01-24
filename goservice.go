package goservice

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
)

type ServiceRunner interface {
	// Runs the service. Blocks until it is finished.
	Run() error
	// Starts the service. Should return quickly and leave
	// the service running in the background.
	Start() error
	// Stops the service.
	Stop() error
}

type GoService struct {
	Service service.Service
	adapter *serviceAdapter
}

func MakeServiceFunction(config service.Config, fn ServiceFunction, loggingConfig *LoggingConfig, controllerName string, pollDuration time.Duration) *GoService {
	return MakeService(config, MakeController(controllerName, fn, pollDuration), loggingConfig)
}

func MakeService(config service.Config, runner ServiceRunner, loggingConfig *LoggingConfig) *GoService {
	adapter := newServiceAdapter(runner, loggingConfig)
	svc, err := service.New(adapter, &config)
	if err != nil {
		logAndQuit(err.Error(), true)
	}
	return &GoService{svc, adapter}
}

func (gs *GoService) Start() {
	if service.Interactive() {
		gs.adapter.Start(gs.Service)
	} else {
		err := gs.Service.Run()
		if err != nil {
			logAndQuit(err.Error(), true)
		}
	}
}

func (gs *GoService) Stop() {
	if service.Interactive() {
		gs.adapter.Stop(gs.Service)
	} else {
		err := gs.Service.Stop()
		if err != nil {
			logAndQuit(err.Error(), true)
		}
	}
}

// Utility function to handle Ctrl+C keyboard presses, if required.
func handleCtrlC(response func()) {
	// Handle CTRL+C, try for a nice clean shutdown.
	keyboardQuitHandler := make(chan os.Signal, 1)
	signal.Notify(keyboardQuitHandler, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-keyboardQuitHandler
		response()
	}()
}

func logAndQuit(msg string, failed bool) {
	logAndExit := func(logFn func(_ logrus.FieldLogger, args ...interface{}), exitCode int) {
		logFn(logger, msg)
		os.Exit(exitCode)
	}
	if failed {
		logAndExit(logrus.FieldLogger.Error, 1)
	} else {
		logAndExit(logrus.FieldLogger.Info, 0)
	}
}
