package pubsub_test

import (
	"fmt"
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

func TestPSubscribe(t *testing.T) {
	t.Parallel()

	received := make(chan interface{}, 10)
	c := &pubsub.Client{ID: "1", Callback: func(msg interface{}) {
		received <- msg
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.PSubscribe(c, "abci.Invoices.*")

	s.Publish("Tombstone", "NewBlock", "abci.Invoices.Ivan.22")
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

func TestPUnsubscribe(t *testing.T) {
	t.Parallel()

	received := make(chan interface{}, 10)
	c := &pubsub.Client{ID: "1", Callback: func(msg interface{}) {
		received <- msg
	}}

	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	s.PSubscribe(c, "abci.Invoices.*")

	s.Publish("Tombstone", "NewBlock", "abci.Invoices.Ivan.22")
	assert.Equal(t, "Tombstone", <-received)
	assert.Zero(t, len(received))

	s.PUnsubscribe(c, "abci.Invoices.*")

	s.Publish("Toad Men", "NewBlock", "abci.Invoices.Ivan.22")
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
	s.PSubscribe(c, "abci.Invoices.*")

	s.Publish("Tombstone", "NewBlock", "abci.Invoices.Ivan.22")
	assert.Equal(t, "Tombstone", <-received)
	assert.Zero(t, len(received))

	s.UnsubscribeAll(c)

	s.Publish("Toad Men", "NewBlock", "abci")
	assert.Zero(t, len(received))
	s.Publish("Toad Men", "NewBlock", "abci.Invoices.Ivan", "abci.Invoices.Ivan.22")
	assert.Zero(t, len(received))
}

func Benchmark10Clients(b *testing.B)   { benchmarkNClients(10, b) }
func Benchmark100Clients(b *testing.B)  { benchmarkNClients(100, b) }
func Benchmark1000Clients(b *testing.B) { benchmarkNClients(1000, b) }

func benchmarkNClients(n int, b *testing.B) {
	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	for i := 0; i < n; i++ {
		c := &pubsub.Client{ID: fmt.Sprintf("#%d", i), Callback: func(msg interface{}) {}}
		s.Subscribe(c, "abci", "abci.Account", "abci.Invoices")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Publish("Gamora", "NewBlock", "abci", "abci.Account", "abci.Invoices")
	}
}

func Benchmark100ClientsUsingPatterns(b *testing.B) {
	s := pubsub.NewServer(1)
	go s.Start()
	defer s.Stop()

	for i := 0; i < 100; i++ {
		c := &pubsub.Client{ID: fmt.Sprintf("#%d", i), Callback: func(msg interface{}) {}}
		s.PSubscribe(c, "abci.Invoices.*")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Publish("Hydro-Man", "NewBlock", "abci.Account", "abci.Invoices.Ivan")
	}
}
