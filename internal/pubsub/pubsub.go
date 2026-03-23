package pubsub

import (
	"sync"
)

// PubSub is a simple in-memory pub/sub messaging system.
type PubSub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan string
}

// Subscription represents an active subscription to a channel.
type Subscription struct {
	Ch   <-chan string
	stop func()
}

// Unsubscribe closes the subscription and stops receiving messages.
func (s *Subscription) Unsubscribe() {
	if s.stop != nil {
		s.stop()
		s.stop = nil
	}
}

// New creates a new PubSub.
func New() *PubSub {
	return &PubSub{
		subscribers: make(map[string][]chan string),
	}
}

// Subscribe subscribes to a channel and returns a Subscription.
func (p *PubSub) Subscribe(channel string) *Subscription {
	ch := make(chan string, 100)
	p.mu.Lock()
	p.subscribers[channel] = append(p.subscribers[channel], ch)
	p.mu.Unlock()
	return &Subscription{
		Ch: ch,
		stop: func() {
			p.Unsubscribe(channel, ch)
		},
	}
}

// Publish sends a message to all subscribers of the channel (non-blocking).
func (p *PubSub) Publish(channel, message string) {
	p.mu.RLock()
	subs := make([]chan string, len(p.subscribers[channel]))
	copy(subs, p.subscribers[channel])
	p.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- message:
		default:
			// channel full, skip to avoid blocking
		}
	}
}

// Unsubscribe removes a subscriber from a channel and closes the channel.
// The ch parameter must be the channel returned by Subscribe.
func (p *PubSub) Unsubscribe(channel string, ch chan string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	subs := p.subscribers[channel]
	for i, c := range subs {
		if c == ch {
			p.subscribers[channel] = append(subs[:i], subs[i+1:]...)
			close(c)
			if len(p.subscribers[channel]) == 0 {
				delete(p.subscribers, channel)
			}
			return
		}
	}
}
