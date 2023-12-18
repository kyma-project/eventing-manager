package signals

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	// onlyOneSignalHandler to make sure that only one signal handler is registered.
	//nolint:gochecknoglobals // needs to be global so that the application is always closed on second signal
	onlyOneSignalHandler = make(chan struct{})

	ErrTerminationRequested = errors.New("received a termination signal")
)

// shutdownSignals array of system signals to cause shutdown.
func shutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
}

// SetupSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupSignalHandler() <-chan struct{} {
	close(onlyOneSignalHandler) // panics when called twice

	return setupStopChannel()
}

func setupStopChannel() <-chan struct{} {
	const chanSize = 2 // we want to be able to read twice
	stop := make(chan struct{})
	osSignal := make(chan os.Signal, chanSize)
	signal.Notify(osSignal, shutdownSignals()...)
	go func() {
		<-osSignal
		close(stop)
		<-osSignal
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

// signalContext represents a signal context.
type signalContext struct {
	stopCh <-chan struct{}
}

// NewContext creates a new singleton context with SetupSignalHandler()
// as our Done() channel. This method can be called only once.
func NewContext() context.Context {
	return &signalContext{stopCh: SetupSignalHandler()}
}

// NewReusableContext creates a new context with setupStopChannel() as our Done() channel.
// This method can be called multiple times, returning new contexts.
func NewReusableContext() context.Context {
	return &signalContext{stopCh: setupStopChannel()}
}

// Deadline implements context.Context.
func (scc *signalContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

// Done implements context.Context.
func (scc *signalContext) Done() <-chan struct{} {
	return scc.stopCh
}

// Err implements context.Context.
func (scc *signalContext) Err() error {
	select {
	case _, ok := <-scc.Done():
		if !ok {
			return ErrTerminationRequested
		}
	default:
	}
	return nil
}

// Value implements context.Context.
func (scc *signalContext) Value(interface{}) interface{} {
	return nil
}
