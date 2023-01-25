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

type ServiceRunnerProvider = func(logger *logrus.Logger) (ServiceRunner, error)

type GoService struct {
	Service service.Service
	Logger  *logrus.Logger
	adapter *serviceAdapter
}

func MakeServiceFunction(config service.Config, fn ServiceFunction, loggingConfig *LoggingConfig, controllerName string, pollDuration time.Duration) *GoService {
	return MakeService(config, func(logger *logrus.Logger) (ServiceRunner, error) {
		return MakeController(controllerName, fn, pollDuration, logger), nil
	}, loggingConfig)
}

func MakeService(config service.Config, serviceRunnerProvider ServiceRunnerProvider, loggingConfig *LoggingConfig) *GoService {
	// Our logger. Initially logs to console, but our code will later
	// "mute" the console output after some hooks are added.
	logger := createLogger(os.Stderr, logrus.DebugLevel)
	runner, err := serviceRunnerProvider(logger)
	if err != nil {
		logAndQuit(logger, err.Error(), true)
	}
	adapter := newServiceAdapter(runner, loggingConfig, logger)
	svc, err := service.New(adapter, &config)
	if err != nil {
		logAndQuit(logger, err.Error(), true)
	}
	return &GoService{svc, logger, adapter}
}

func (gs *GoService) Start() {
	if service.Interactive() {
		gs.adapter.Start(gs.Service)
	} else {
		err := gs.Service.Run()
		if err != nil {
			logAndQuit(gs.Logger, err.Error(), true)
		}
	}
}

func (gs *GoService) Stop() {
	if service.Interactive() {
		gs.adapter.Stop(gs.Service)
	} else {
		err := gs.Service.Stop()
		if err != nil {
			logAndQuit(gs.Logger, err.Error(), true)
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

func logAndQuit(logger *logrus.Logger, msg string, failed bool) {
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
