package goservice

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ServiceFunction = func(quit chan bool, hasQuit chan bool) (bool, error)

// A "Controller" is basically a function that can be run in a goroutine
// and can be asked to stop, and will report when it has stopped.
// The "fn" function returns a boolean that indicates if it did anything.
// This is only useful to the poll() function, which will pause (for
// "pollTime") if there was nothing to do, or repeat instantly if there was.
type Controller struct {
	name         string
	fn           ServiceFunction
	pollTime     time.Duration
	stop         chan bool
	hasStopped   chan bool
	hasCompleted chan bool
}

// Creates a named controller from the given function.
func MakeController(name string, fn func(quit chan bool, hasQuit chan bool) (bool, error), pollTime time.Duration) *Controller {
	return &Controller{name, fn, pollTime, make(chan bool, 1), make(chan bool, 1), make(chan bool, 1)}
}

// Signals the controller to stop, then waits for it to end.
func (ctrl *Controller) Stop() error {
	QuitControllersAndWait([]*Controller{ctrl})
	ctrl.hasCompleted <- true
	return nil
}

// Repeatedly runs the function until signalled to stop.
func (ctrl *Controller) Run() error {
	var quit = false
	for !quit {
		againNow, err := ctrl.fn(ctrl.stop, ctrl.hasStopped)
		if err != nil {
			logger.Error(err)
		}
		// If function says "run again now", check first
		// for whether stop has been requested.
		if againNow {
			if len(ctrl.stop) > 0 {
				break
			} else {
				continue
			}
		}
		// Wait for stop signal, or poll time to pass.
		select {
		case quit = <-ctrl.stop:
		case <-time.After(ctrl.pollTime):
		}
	}
	// Notify controller that work loop has stopped
	ctrl.hasStopped <- true
	// Wait for controller to complete. The service framework
	// will perform a hard exit of the program when it thinks
	// that the service has finished (i.e. when this function
	// returns). We still might have some cleanup to do though.
	<-ctrl.hasCompleted
	ctrl.close()
	return nil
}

func (ctrl *Controller) Start() error {
	go ctrl.Run()
	return nil
}

// Closes the channels associated with this controller.
func (ctrl *Controller) close() {
	close(ctrl.stop)
	close(ctrl.hasStopped)
	close(ctrl.hasCompleted)
}

func (ctrl *Controller) AsLogFields() logrus.Fields {
	return logrus.Fields{"controller": ctrl.name}
}

// Asks all the given controllers to stop, and waits until they have all stopped.
func QuitControllersAndWait(controllers []*Controller) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(controllers))
	for _, ctrl := range controllers {
		logger.WithFields(ctrl.AsLogFields()).Info("Controller is stopping")
		ctrl.stop <- true
	}
	for _, ctrl := range controllers {
		go func(q *Controller) {
			defer waitGroup.Done()
			<-q.hasStopped
			logger.WithFields(q.AsLogFields()).Info("Controller has stopped")
		}(ctrl)
	}
	waitGroup.Wait()
}
