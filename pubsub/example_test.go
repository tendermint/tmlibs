package pubsub_test

import (
	"testing"
	"time"

	"github.com/tendermint/tmlibs/pubsub"
)

func TestExample(t *testing.T) {
	received := false
	c := &pubsub.Client{ID: "1", MsgType: "NewBlock", Callback: func(msg interface{}) {
		received = true
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.Subscribe(c, "abci")
	s.Publish("hello", "NewBlock", "abci")
	time.Sleep(10 * time.Millisecond)
	if !received {
		t.Fatal("Client did not receive a message")
	}
}
