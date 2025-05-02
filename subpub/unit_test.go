package subpub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubscribePublish(t *testing.T) {
	bus := NewSubPub()
	defer func() {
		_ = bus.Close(context.Background())
	}()

	var wg sync.WaitGroup
	var received []string
	var mu sync.Mutex

	sub, err := bus.Subscribe("test", func(msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, msg.(string))
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	wg.Add(1)
	if err := bus.Publish("test", "message1"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	waitWithTimeout(&wg, 1*time.Second, t)

	mu.Lock()
	if len(received) != 1 || received[0] != "message1" {
		t.Errorf("Expected ['message1'], got %v", received)
	}
	mu.Unlock()
}

func TestUnsubscribe(t *testing.T) {
	bus := NewSubPub()
	defer func() {
		_ = bus.Close(context.Background())
	}()

	var received []string
	var mu sync.Mutex

	sub, _ := bus.Subscribe("test", func(msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, msg.(string))
	})

	sub.Unsubscribe()

	if err := bus.Publish("test", "message1"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(received) > 0 {
		t.Errorf("Expected no messages after unsubscribe, got %v", received)
	}
	mu.Unlock()
}

func TestClose(t *testing.T) {
	bus := NewSubPub()

	var wg sync.WaitGroup
	wg.Add(1)

	_, _ = bus.Subscribe("test", func(msg interface{}) {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
	})

	_ = bus.Publish("test", "message1")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := bus.Close(ctx)
	if err == nil {
		t.Error("Expected context canceled error, got nil")
	}

	wg.Wait()
}

func TestConcurrentPublish(t *testing.T) {
	bus := NewSubPub()
	defer func() {
		_ = bus.Close(context.Background())
	}()

	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	_, _ = bus.Subscribe("test", func(msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		counter++
		wg.Done()
	})

	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			_ = bus.Publish("test", "message")
		}()
	}

	waitWithTimeout(&wg, 1*time.Second, t)

	mu.Lock()
	if counter != 100 {
		t.Errorf("Expected 100 messages, got %d", counter)
	}
	mu.Unlock()
}

func TestMessageOrder(t *testing.T) {
	bus := NewSubPub()
	defer func() {
		_ = bus.Close(context.Background())
	}()

	var received []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	_, _ = bus.Subscribe("order", func(msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, msg.(int))
		wg.Done()
	})

	wg.Add(10)
	for i := 0; i < 10; i++ {
		_ = bus.Publish("order", i)
		time.Sleep(time.Millisecond * 100)
	}

	waitWithTimeout(&wg, 1*time.Second, t)

	mu.Lock()
	defer mu.Unlock()
	for i := 0; i < 10; i++ {
		if received[i] != i {
			t.Errorf("Message out of order at %d: got %d", i, received[i])
		}
	}
}

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration, t *testing.T) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Fatal("Timeout waiting for operation")
	}
}
