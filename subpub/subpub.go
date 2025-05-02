package subpub

import (
	"context"
	"sync"
)

type message struct {
	subject string
	msg     interface{}
}

type subPub struct {
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
	subscribers map[string][]*subscription
	messageChan chan message
	shutdown    chan struct{}
}

type subscription struct {
	subject     string
	handler     MessageHandler
	unsubscribe chan struct{}
}

func (sp *subPub) Subscribe(subject string, sb MessageHandler) (Subscription, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sub := &subscription{
		subject:     subject,
		handler:     sb,
		unsubscribe: make(chan struct{}),
	}

	sp.subscribers[subject] = append(sp.subscribers[subject], sub)

	return sub, nil
}

func (sp *subPub) Publish(subject string, msg interface{}) error {
	pubMsg := message{subject: subject, msg: msg}
	select {
	case sp.messageChan <- pubMsg:
		return nil
	case <-sp.shutdown:
		return context.Canceled
	}
}

func (sp *subPub) Close(ctx context.Context) error {
	close(sp.shutdown)

	done := make(chan struct{})
	go func() {
		sp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (sub subscription) Unsubscribe() {
	close(sub.unsubscribe)
}

func (sp *subPub) removeSubscription(subject string, sub *subscription) {
	subs := sp.subscribers[subject]
	for i, s := range subs {
		if s == sub {
			sp.subscribers[subject][i] = sp.subscribers[subject][len(sp.subscribers[subject])-1]
			sp.subscribers[subject] = subs[:len(subs)-1]
			break
		}
	}
}

func newSubPub() SubPub {
	sp := &subPub{
		mu:          new(sync.RWMutex),
		wg:          new(sync.WaitGroup),
		subscribers: make(map[string][]*subscription),
		messageChan: make(chan message),
		shutdown:    make(chan struct{}),
	}

	sp.wg.Add(1)
	sp.start()

	return sp
}

func (sp *subPub) start() {
	defer sp.wg.Done()

	for {
		select {
		case msg := <-sp.messageChan:
			sp.mu.RLock()
			subs := sp.subscribers[msg.subject]
			sp.mu.RUnlock()

			for _, sub := range subs {
				select {
				case <-sub.unsubscribe:
					sp.mu.Lock()
					sp.removeSubscription(msg.subject, sub)
					sp.mu.Unlock()
				default:
					sp.wg.Add(1)
					go func(s *subscription, m interface{}) {
						defer sp.wg.Done()
						s.handler(m)
					}(sub, msg.msg)
				}
			}

		case <-sp.shutdown:
			return
		}
	}
}
