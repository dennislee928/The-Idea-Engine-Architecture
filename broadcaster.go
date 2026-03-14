package main

import "sync"

type Broadcaster struct {
	mu          sync.RWMutex
	nextID      int
	subscribers map[int]chan DBInsight
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[int]chan DBInsight),
	}
}

func (b *Broadcaster) Subscribe() (int, <-chan DBInsight) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	ch := make(chan DBInsight, 16)
	b.subscribers[id] = ch
	return id, ch
}

func (b *Broadcaster) Unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subscribers[id]; ok {
		delete(b.subscribers, id)
		close(ch)
	}
}

func (b *Broadcaster) Broadcast(insight DBInsight) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- insight:
		default:
		}
	}
}
