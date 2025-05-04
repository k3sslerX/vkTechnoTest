package service

import (
	"context"
	_ "errors"
	"log"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"subpub"
	pb "subscribeservice/proto/pubsub"
)

type PubSubService struct {
	pb.UnimplementedPubSubServer
	bus    subpub.SubPub
	logger *log.Logger
	mu     sync.Mutex
	subs   map[string]map[chan string]struct{}
}

func NewPubSubService(bus subpub.SubPub, logger *log.Logger) *PubSubService {
	return &PubSubService{
		bus:    bus,
		logger: logger,
		subs:   make(map[string]map[chan string]struct{}),
	}
}

func (s *PubSubService) Subscribe(req *pb.SubscribeRequest, stream pb.PubSub_SubscribeServer) error {
	key := req.GetKey()
	if key == "" {
		return status.Error(codes.InvalidArgument, "key cannot be empty")
	}

	eventChan := make(chan string, 100)
	s.addSubscriber(key, eventChan, stream)
	defer s.removeSubscriber(key, eventChan)

	s.logger.Printf("New subscription for key: %s", key)

	for {
		select {
		case data, ok := <-eventChan:
			if !ok {
				return nil
			}
			if err := stream.Send(&pb.Event{Data: data}); err != nil {
				s.logger.Printf("Stream send error: %v", err)
				return status.Error(codes.Internal, "failed to send event")
			}
		case <-stream.Context().Done():
			s.logger.Printf("Subscription ended for key: %s", key)
			return nil
		}
	}
}

func (s *PubSubService) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	key := req.GetKey()
	data := req.GetData()

	if key == "" || data == "" {
		return nil, status.Error(codes.InvalidArgument, "key and data cannot be empty")
	}

	if err := s.bus.Publish(key, data); err != nil {
		s.logger.Printf("Publish error: %v", err)
		return nil, status.Error(codes.Internal, "publish failed")
	}

	s.logger.Printf("Published to key %s: %s", key, data)
	return &emptypb.Empty{}, nil
}

func (s *PubSubService) addSubscriber(key string, ch chan string, stream pb.PubSub_SubscribeServer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subs[key]; !ok {
		s.subs[key] = make(map[chan string]struct{})
		sub, _ := s.bus.Subscribe(key, func(msg interface{}) {
			s.notifySubscribers(key, msg.(string))
		})
		go func() {
			<-stream.Context().Done()
			sub.Unsubscribe()
		}()
	}
	s.subs[key][ch] = struct{}{}
}

func (s *PubSubService) removeSubscriber(key string, ch chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if subs, ok := s.subs[key]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(s.subs, key)
		}
	}
}

func (s *PubSubService) notifySubscribers(key, data string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if subs, ok := s.subs[key]; ok {
		for ch := range subs {
			select {
			case ch <- data:
			default:
				s.logger.Printf("Subscriber channel full, dropping message for key: %s", key)
			}
		}
	}
}
