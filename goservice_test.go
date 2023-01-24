package goservice

import (
	"fmt"
	"testing"
	"time"

	"github.com/kardianos/service"
)

func TestService(t *testing.T) {
	t.Parallel()
	serviceConfig := service.Config{
		DisplayName: "My Test Service",
		Name:        "MyTestService",
		Description: "A test service",
	}
	loggingConfig := &LoggingConfig{
		Debug:            true,
		LogFilename:      "testLog.log",
		ErrorLogFilename: "testErrorLog.log",
	}
	iterations := 0
	serviceFunction := func(quit chan bool, hasQuit chan bool) (bool, error) {
		iterations = iterations + 1
		var err error = nil
		if iterations == 3 {
			err = fmt.Errorf("Simulated error")
		}
		return true, err
	}
	runner := MakeController("TestController", serviceFunction, time.Second*1)
	svc := MakeService(serviceConfig, runner, loggingConfig)
	go func() {
		time.Sleep(time.Second * 5)
		svc.Stop()
	}()
	svc.Start()
}
