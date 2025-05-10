package subpub

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewSubPub(t *testing.T) {
	sp := NewSubPub()
	if sp == nil {
		t.Fatal("NewSubPub() returned nil")
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	var receivedMsg interface{}
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(msg interface{}) {
		receivedMsg = msg
		wg.Done()
	}

	sub, err := sp.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	err = sp.Publish("test", "hello")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()
	if receivedMsg != "hello" {
		t.Errorf("Expected 'hello', got %v", receivedMsg)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	var results []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)

	handler1 := func(msg interface{}) {
		mu.Lock()
		results = append(results, "handler1:"+msg.(string))
		mu.Unlock()
		wg.Done()
	}

	handler2 := func(msg interface{}) {
		mu.Lock()
		results = append(results, "handler2:"+msg.(string))
		mu.Unlock()
		wg.Done()
	}

	sub1, err := sp.Subscribe("test", handler1)
	if err != nil {
		t.Fatalf("Subscribe 1 failed: %v", err)
	}
	defer sub1.Unsubscribe()

	sub2, err := sp.Subscribe("test", handler2)
	if err != nil {
		t.Fatalf("Subscribe 2 failed: %v", err)
	}
	defer sub2.Unsubscribe()

	err = sp.Publish("test", "message")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
}

func TestUnsubscribe(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	var received bool
	handler := func(msg interface{}) {
		received = true
	}

	sub, err := sp.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	sub.Unsubscribe()

	err = sp.Publish("test", "hello")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if received {
		t.Error("Handler was called after unsubscribe")
	}
}

func TestPublishToClosed(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	ctx := context.Background()
	err := sp.Close(ctx)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	err = sp.Publish("test", "hello")
	if err == nil {
		t.Error("Expected error when publishing to closed subpub")
	}
}

func TestSubscribeToClosed(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	ctx := context.Background()
	err := sp.Close(ctx)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_, err = sp.Subscribe("test", func(msg interface{}) {})
	if err == nil {
		t.Error("Expected error when subscribing to closed subpub")
	}
}

func TestSlowSubscriber(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	var fastReceived, slowReceived bool
	var wg sync.WaitGroup
	wg.Add(2)

	fastHandler := func(msg interface{}) {
		fastReceived = true
		wg.Done()
	}

	slowHandler := func(msg interface{}) {
		time.Sleep(100 * time.Millisecond)
		slowReceived = true
		wg.Done()
	}

	fastSub, err := sp.Subscribe("test", fastHandler)
	if err != nil {
		t.Fatalf("Fast subscribe failed: %v", err)
	}
	defer fastSub.Unsubscribe()

	slowSub, err := sp.Subscribe("test", slowHandler)
	if err != nil {
		t.Fatalf("Slow subscribe failed: %v", err)
	}
	defer slowSub.Unsubscribe()

	err = sp.Publish("test", "message")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()

	if !fastReceived || !slowReceived {
		t.Errorf("Fast received: %v, Slow received: %v", fastReceived, slowReceived)
	}
}

func TestChannelFull(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	sub := &subscription{
		subject:  "test",
		handler:  func(msg interface{}) {},
		msgChan:  make(chan interface{}, 1),
		stopChan: make(chan struct{}),
		pub:      sp,
	}

	sp.mu.Lock()
	sp.subscribers["test"] = []*subscription{sub}
	sp.mu.Unlock()

	err := sp.Publish("test", "msg1")
	if err != nil {
		t.Fatalf("First publish failed: %v", err)
	}

	err = sp.Publish("test", "msg2")
	if err == nil {
		t.Error("Expected error when channel is full")
	}
}

func TestCloseWithTimeout(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	var wg sync.WaitGroup
	wg.Add(1)
	_, err := sp.Subscribe("test", func(msg interface{}) {
		time.Sleep(200 * time.Millisecond)
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	err = sp.Publish("test", "message")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = sp.Close(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}

	wg.Wait()
}

func TestConcurrentPublishSubscribe(t *testing.T) {
	sp := NewSubPub().(*subPublisher)
	var wg sync.WaitGroup
	count := 100

	wg.Add(count * 2)

	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			_, err := sp.Subscribe("test", func(msg interface{}) {})
			if err != nil {
				t.Errorf("Subscribe error: %v", err)
			}
		}()
	}

	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			err := sp.Publish("test", "message")
			if err != nil {
				t.Errorf("Publish error: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestMultipleSubjects(t *testing.T) {
	sp := NewSubPub().(*subPublisher)

	var sub1Received, sub2Received bool
	var wg sync.WaitGroup
	wg.Add(2)

	sub1, err := sp.Subscribe("subject1", func(msg interface{}) {
		sub1Received = true
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Subscribe 1 failed: %v", err)
	}
	defer sub1.Unsubscribe()

	sub2, err := sp.Subscribe("subject2", func(msg interface{}) {
		sub2Received = true
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Subscribe 2 failed: %v", err)
	}
	defer sub2.Unsubscribe()

	err = sp.Publish("subject1", "message1")
	if err != nil {
		t.Fatalf("Publish 1 failed: %v", err)
	}

	err = sp.Publish("subject2", "message2")
	if err != nil {
		t.Fatalf("Publish 2 failed: %v", err)
	}

	wg.Wait()

	if !sub1Received || !sub2Received {
		t.Errorf("sub1Received: %v, sub2Received: %v", sub1Received, sub2Received)
	}
}
