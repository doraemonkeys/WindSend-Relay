package tool

import (
	"sync"
)

type Subscriber[T any] interface {
	OnEvent(event T)
	GetID() string
}

type Publisher[T any] struct {
	mu sync.RWMutex
	// SubscriberID -> Subscriber
	subscribers map[string]Subscriber[T]
}

func NewPublisher[T any]() *Publisher[T] {
	return &Publisher[T]{
		subscribers: make(map[string]Subscriber[T]),
	}
}

func (p *Publisher[T]) Subscribe(subscriber Subscriber[T]) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.subscribers[subscriber.GetID()] = subscriber
}

func (p *Publisher[T]) Unsubscribe(subscriberID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.subscribers, subscriberID)
}

func (p *Publisher[T]) Publish(event T) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, subscriber := range p.subscribers {
		subscriber.OnEvent(event)
	}
}

func (p *Publisher[T]) GetSubscriberCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.subscribers)
}
