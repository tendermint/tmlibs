package common

import (
	"testing"
)

func TestBaseService(t *testing.T) {

	ts := NewBaseService(nil, "TestService", &testServiceCore{})
	ts.Start()

	go func() {
		ts.Stop()
	}()

	for i := 0; i < 10; i++ {
		ts.wait()
	}

}

type testServiceCore struct{}

func (tc *testServiceCore) OnStart() error {
	return nil
}

func (tc *testServiceCore) OnStop() error {
	return nil
}
