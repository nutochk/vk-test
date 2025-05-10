package subpub

import (
	"context"
	"fmt"
	"sync"
)

type MessageHandler func(msg interface{})

type Subscription interface {
	Unsubscribe()
}

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

type subscription struct {
	subject  string
	handler  MessageHandler
	msgChan  chan interface{}
	stopChan chan struct{}
	pub      *subPublisher
}

func (s *subscription) Unsubscribe() {
	s.pub.unsubscribe(s)
}

type subPublisher struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscription
	closed      bool
	wg          sync.WaitGroup
}

func NewSubPub() SubPub {
	return &subPublisher{
		subscribers: make(map[string][]*subscription),
	}
}

func (s *subPublisher) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("subpub is closed")
	}

	sub := &subscription{
		subject:  subject,
		handler:  cb,
		msgChan:  make(chan interface{}, 100),
		stopChan: make(chan struct{}),
		pub:      s,
	}

	s.subscribers[subject] = append(s.subscribers[subject], sub)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case msg := <-sub.msgChan:
				sub.handler(msg)
			case <-sub.stopChan:
				return
			}
		}
	}()

	return sub, nil
}

func (s *subPublisher) unsubscribe(sub *subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs, ok := s.subscribers[sub.subject]
	if !ok {
		return
	}

	for i, v := range subs {
		if v == sub {
			s.subscribers[sub.subject] = append(subs[:i], subs[i+1:]...)
			close(sub.stopChan)
			close(sub.msgChan)
			return
		}
	}
}

func (s *subPublisher) Publish(subject string, msg interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return fmt.Errorf("subpub is closed")
	}

	subs, ok := s.subscribers[subject]
	if !ok {
		return nil
	}

	for _, sub := range subs {
		select {
		case sub.msgChan <- msg:
		default:
			return fmt.Errorf("sud channel is full")
		}
	}

	return nil
}

func (s *subPublisher) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	s.mu.Lock()
	for subject, subs := range s.subscribers {
		for _, sub := range subs {
			close(sub.stopChan)
			close(sub.msgChan)
		}
		delete(s.subscribers, subject)
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
