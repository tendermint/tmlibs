package pubsub_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tmlibs/pubsub"
)

func TestSubscribe(t *testing.T) {
	t.Parallel()

	received := make(chan interface{}, 10)
	c := &pubsub.Client{ID: "1", Callback: func(msg interface{}) {
		received <- msg
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.Subscribe(c, "abci")
	s.Subscribe(c, "abci")
	s.Subscribe(c, "abci.Account")

	// client gets its message
	s.Publish("Tombstone", "NewBlock", "abci")
	assert.Equal(t, "Tombstone", <-received)
	assert.Zero(t, len(received))

	// client gets its message only once per Publish
	s.Publish("Unicorn", "NewBlock", "abci", "abci.Account")
	assert.Equal(t, "Unicorn", <-received)
	assert.Zero(t, len(received))

	// if nobody is subscribed to the channel, nothing happens
	s.Publish("Cypher", "NewBlock", "unknown")
	select {
	case msg := <-received:
		t.Fatalf("Client should not receive a message, got %v", msg)
	case <-time.After(10 * time.Millisecond):
	}
	assert.Zero(t, len(received))
}

func TestClientWithMsgTypeReceivesOnlyEventsOfThatType(t *testing.T) {
	t.Parallel()

	received := make(chan interface{}, 10)
	c := &pubsub.Client{ID: "1", MsgType: "NewBlock", Callback: func(msg interface{}) {
		received <- msg
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.Subscribe(c, "abci")

	s.Publish("Centurions", "NewBlock", "abci")
	assert.Equal(t, "Centurions", <-received)
	assert.Zero(t, len(received))

	s.Publish("Dani Moonstar", "NewBlockHeader", "abci")
	select {
	case msg := <-received:
		t.Fatalf("Client should not receive a message, got %v", msg)
	case <-time.After(10 * time.Millisecond):
	}
	assert.Zero(t, len(received))
}

func TestSubscribePatterns(t *testing.T) {
	t.Parallel()

	received := make(chan interface{}, 10)
	c := &pubsub.Client{ID: "1", Callback: func(msg interface{}) {
		received <- msg
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.SubscribePatterns(c, "abci.Invoices.*")

	s.Publish("Tombstone", "NewBlock", "abci.Invoices.Ivan", "abci.Invoices.Ivan.22")
	assert.Equal(t, "Tombstone", <-received)
	assert.Zero(t, len(received))
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()

	received := make(chan interface{}, 10)
	c := &pubsub.Client{ID: "1", Callback: func(msg interface{}) {
		received <- msg
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.Subscribe(c, "abci")

	s.Publish("Tombstone", "NewBlock", "abci")
	assert.Equal(t, "Tombstone", <-received)
	assert.Zero(t, len(received))

	s.Unsubscribe(c, "abci")

	s.Publish("Thunderball", "NewBlock", "abci")
	assert.Zero(t, len(received))
}

func TestUnsubscribePatterns(t *testing.T) {
	t.Parallel()

	received := make(chan interface{}, 10)
	c := &pubsub.Client{ID: "1", Callback: func(msg interface{}) {
		received <- msg
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.SubscribePatterns(c, "abci.Invoices.*")

	s.Publish("Tombstone", "NewBlock", "abci.Invoices.Ivan", "abci.Invoices.Ivan.22")
	assert.Equal(t, "Tombstone", <-received)
	assert.Zero(t, len(received))

	s.UnsubscribePatterns(c, "abci.Invoices.*")

	s.Publish("Toad Men", "NewBlock", "abci.Invoices.Ivan", "abci.Invoices.Ivan.22")
	assert.Zero(t, len(received))
}

func TestUnsubscribeAll(t *testing.T) {
	t.Parallel()

	received := make(chan interface{}, 10)
	c := &pubsub.Client{ID: "1", Callback: func(msg interface{}) {
		received <- msg
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.Subscribe(c, "abci")
	s.SubscribePatterns(c, "abci.Invoices.*")

	s.Publish("Tombstone", "NewBlock", "abci.Invoices.Ivan", "abci.Invoices.Ivan.22")
	assert.Equal(t, "Tombstone", <-received)
	assert.Zero(t, len(received))

	s.UnsubscribeAll(c)

	s.Publish("Toad Men", "NewBlock", "abci")
	assert.Zero(t, len(received))
	s.Publish("Toad Men", "NewBlock", "abci.Invoices.Ivan", "abci.Invoices.Ivan.22")
	assert.Zero(t, len(received))
}
