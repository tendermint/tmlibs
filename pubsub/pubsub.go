package pubsub

import "path/filepath"

// Client listens for messages.
type Client struct {
	ID       string                // unique identifier (NOTE: must be unique)
	MsgType  interface{}           // msg type client interested in (nil means we don't care)
	Callback func(msg interface{}) // callback which will be called upon message receiving

	channels []string // channels client interested in
	patterns []string // patterns client interested in
}

type operation int

const (
	sub operation = iota
	pub
	unsub
	unsubAll
	shutdown
)

type cmd struct {
	op       operation
	channels []string
	patterns []string
	client   *Client
	msg      interface{}
	msgType  interface{}
}

// Server allows clients to subscribe/unsubscribe to channels and patterns and
// manages internal state.
type Server struct {
	cmds     chan cmd
	capacity int
}

// NewServer returns a new server.
func NewServer(capacity int) *Server {
	return &Server{make(chan cmd), capacity}
}

// Subscribe client to channels.
func (s *Server) Subscribe(c *Client, channels ...string) {
	s.cmds <- cmd{op: sub, channels: channels, client: c}
}

// SubscribePatterns client to patterns.
func (s *Server) SubscribePatterns(c *Client, patterns ...string) {
	s.cmds <- cmd{op: sub, patterns: patterns, client: c}
}

// Unsubscribe client from channels.
func (s *Server) Unsubscribe(c *Client, channels ...string) {
	if len(channels) > 0 {
		s.cmds <- cmd{op: unsub, channels: channels, client: c}
	}
}

// UnsubscribePatterns client from patterns.
func (s *Server) UnsubscribePatterns(c *Client, patterns ...string) {
	if len(patterns) > 0 {
		s.cmds <- cmd{op: unsub, patterns: patterns, client: c}
	}
}

// UnsubscribeAll client from everything.
func (s *Server) UnsubscribeAll(c *Client) {
	s.cmds <- cmd{op: unsubAll, client: c}
}

// Publish message msg of type msgType onto channels.
func (s *Server) Publish(msg interface{}, msgType interface{}, channels ...string) {
	s.cmds <- cmd{op: pub, channels: channels, msg: msg, msgType: msgType}
}

// Stop shutdowns the server.
func (s *Server) Stop() {
	s.cmds <- cmd{op: shutdown}
}

// Start creates a main loop. Only exits when shutdown operation is received.
func (s *Server) Start() {
	state := state{
		channels: make(map[string]map[string]struct{}),
		patterns: make(map[string]map[string]struct{}),
		clients:  make(map[string]*Client),
	}

loop:
	for cmd := range s.cmds {
		sentTo := make(map[string]bool)

		if cmd.channels == nil && cmd.patterns == nil {
			switch cmd.op {
			case unsubAll:
				state.removeClient(state.clients[cmd.client.ID])
			case shutdown:
				break loop
			}
			continue loop
		}

		for _, channel := range cmd.channels {
			switch cmd.op {
			case sub:
				state.add(channel, false, cmd.client)
			case pub:
				state.send(channel, cmd.msg, cmd.msgType, sentTo)
			case unsub:
				state.remove(channel, false, cmd.client)
			}
		}

		for _, pattern := range cmd.patterns {
			switch cmd.op {
			case sub:
				state.add(pattern, true, cmd.client)
			case unsub:
				state.remove(pattern, true, cmd.client)
			}
		}
	}
}

// NOTE: not goroutine safe
type state struct {
	channels map[string]map[string]struct{}
	patterns map[string]map[string]struct{}
	clients  map[string]*Client
}

func (state *state) add(s string, isPattern bool, c *Client) {
	state.clients[c.ID] = c

	if isPattern {
		if state.patterns[s] == nil {
			state.patterns[s] = make(map[string]struct{})
		}
		state.patterns[s][c.ID] = struct{}{}
		state.clients[c.ID].patterns = appendIfMissing(state.clients[c.ID].patterns, s)
	} else {
		if state.channels[s] == nil {
			state.channels[s] = make(map[string]struct{})
		}
		state.channels[s][c.ID] = struct{}{}
		state.clients[c.ID].channels = appendIfMissing(state.clients[c.ID].channels, s)
	}
}

func (state *state) send(channel string, msg interface{}, msgType interface{}, sentTo map[string]bool) {
	for clientID := range state.channels[channel] {
		if sentTo[clientID] {
			continue
		}
		if state.clients[clientID].MsgType == nil || state.clients[clientID].MsgType == msgType {
			sentTo[clientID] = true
			state.clients[clientID].Callback(msg)
		}
	}

	pattern, ok := match(state.patterns, channel)
	if ok {
		for clientID := range state.patterns[pattern] {
			if sentTo[clientID] {
				continue
			}
			if state.clients[clientID].MsgType == nil || state.clients[clientID].MsgType == msgType {
				sentTo[clientID] = true
				state.clients[clientID].Callback(msg)
			}
		}
	}
}

func (state *state) removeClient(c *Client) {
	for _, s := range c.channels {
		delete(state.channels[s], c.ID)
	}
	for _, s := range c.patterns {
		delete(state.patterns[s], c.ID)
	}
	delete(state.clients, c.ID)
}

func (state *state) remove(s string, isPattern bool, c *Client) {
	if _, ok := state.clients[c.ID]; !ok {
		return
	}

	if isPattern {
		if _, ok := state.patterns[s]; !ok {
			return
		}
		if _, ok := state.patterns[s][c.ID]; !ok {
			return
		}
		delete(state.patterns[s], c.ID)
		for i, p := range state.clients[c.ID].patterns {
			if p == s {
				state.clients[c.ID].patterns[i] = state.clients[c.ID].patterns[len(state.clients[c.ID].patterns)-1]
				state.clients[c.ID].patterns = state.clients[c.ID].patterns[:len(state.clients[c.ID].patterns)-1]
				break
			}
		}
	} else {
		if _, ok := state.channels[s]; !ok {
			return
		}
		if _, ok := state.channels[s][c.ID]; !ok {
			return
		}
		delete(state.channels[s], c.ID)
		for i, ch := range state.clients[c.ID].channels {
			if ch == s {
				state.clients[c.ID].channels[i] = state.clients[c.ID].channels[len(state.clients[c.ID].channels)-1]
				state.clients[c.ID].channels = state.clients[c.ID].channels[:len(state.clients[c.ID].channels)-1]
				break
			}
		}
	}

	if len(state.channels[s]) == 0 {
		delete(state.channels, s)
	}
	if len(state.patterns[s]) == 0 {
		delete(state.patterns, s)
	}

	if state.clients[c.ID].NoSubscriptions() {
		delete(state.clients, c.ID)
	}
}

func appendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func match(patterns map[string]map[string]struct{}, channel string) (pattern string, ok bool) {
	for pattern := range patterns {
		// TODO: is filepath ok? If not we must replace it with something else
		if matched, _ := filepath.Match(pattern, channel); matched {
			return pattern, true
		}
	}
	return "", false
}

// NoSubscriptions returns true if client have no subscriptions.
func (c *Client) NoSubscriptions() bool {
	return len(c.channels) == 0 && len(c.patterns) == 0
}
