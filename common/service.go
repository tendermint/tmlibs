package common

import (
	"sync/atomic"

	"github.com/tendermint/tmlibs/log"
)

// ServiceCore is a service that is implemented by a user.
type ServiceCore interface {
	OnStart() error
	OnStop() error
}

// Service represents a way to start, stop and get the status of some service. BaseService is the
// default implementation and should be used by most users.
// SetServiceCore allows a user to set a ServiceCore. Users should implement their logic using
// ServiceCore.
type Service interface {
	Start() error
	Stop() error
	IsRunning() bool
	SetServiceCore(ServiceCore)
	SetLogger(log.Logger)
	String() string
}

/*
Services can be started and then stopped.

Users provide an implementation of ServiceCore and BaseService guarantees that OnStart and OnStop
are called at most once.
Starting an already started service will panic.
Stopping an already stopped (or non-started) service will panic.

Usage:

	// Implement ServiceCore through OnStart() and OnStop().
	type FooServiceCore struct {
		// private fields
	}

	func (fs *FooServiceCore) OnStart() error {
		// initialize private fields
		// start subroutines, etc.
	}

	func (fs *FooServiceCore) OnStop() error {
		// close/destroy private fields
		// stop subroutines, etc.
	}

	fs := NewBaseService(nil, "MyAwesomeService", &FooServiceCore{})
	fs.Start() // this calls OnStart()
	fs.Stop() // this calls OnStop()
*/

// BaseService provides the guarantees that a ServiceCore can only be started and stopped once.
type BaseService struct {
	logger  log.Logger
	name    string
	started uint32 // atomic
	stopped uint32 // atomic
	quit    chan struct{}

	// The "subclass" of BaseService
	impl ServiceCore
}

// NewBaseService returns a base service that wraps an implementation of ServiceCore and handles
// starting and stopping.
func NewBaseService(logger log.Logger, name string, impl ServiceCore) *BaseService {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	return &BaseService{
		logger: logger,
		name:   name,
		quit:   make(chan struct{}),
		impl:   impl,
	}
}

// Start implements Service
func (bs *BaseService) Start() (bool, error) {
	if atomic.CompareAndSwapUint32(&bs.started, 0, 1) {
		if atomic.LoadUint32(&bs.stopped) == 1 {
			bs.logger.Error(Fmt("Not starting %v -- already stopped", bs.name), "impl", bs.impl)
			return false, nil
		} else {
			bs.logger.Info(Fmt("Starting %v", bs.name), "impl", bs.impl)
		}
		err := bs.impl.OnStart()
		if err != nil {
			// revert flag
			atomic.StoreUint32(&bs.started, 0)
			return false, err
		}
		return true, err
	} else {
		bs.logger.Debug(Fmt("Not starting %v -- already started", bs.name), "impl", bs.impl)
		return false, nil
	}
}

// Stop implements Service
func (bs *BaseService) Stop() bool {
	if atomic.CompareAndSwapUint32(&bs.stopped, 0, 1) {
		bs.logger.Info(Fmt("Stopping %v", bs.name), "impl", bs.impl)
		bs.impl.OnStop()
		close(bs.quit)
		return true
	} else {
		bs.logger.Debug(Fmt("Stopping %v (ignoring: already stopped)", bs.name), "impl", bs.impl)
		return false
	}
}

// IsRunning implements Service
func (bs *BaseService) IsRunning() bool {
	return atomic.LoadUint32(&bs.started) == 1 && atomic.LoadUint32(&bs.stopped) == 0
}

// SetServiceCore impleents SetServiceCore
func (bs *BaseService) SetServiceCore(service ServiceCore) {
	bs.impl = service
}

// SetLogger implements Service
func (bs *BaseService) SetLogger(l log.Logger) {
	bs.logger = l
}

// String implements Service
func (bs *BaseService) String() string {
	return bs.name
}

func (bs *BaseService) wait() {
	<-bs.quit
}
