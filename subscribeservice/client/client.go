package client

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "subscribeservice/proto/pubsub"
)

type PubSubClient struct {
	conn    *grpc.ClientConn
	client  pb.PubSubClient
	timeout time.Duration
}

func NewClient(serverAddr string, timeoutSec int) (*PubSubClient, error) {
	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &PubSubClient{
		conn:    conn,
		client:  pb.NewPubSubClient(conn),
		timeout: time.Duration(timeoutSec) * time.Second,
	}, nil
}

func (c *PubSubClient) Close() error {
	return c.conn.Close()
}

func (c *PubSubClient) Publish(ctx context.Context, key, data string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.Publish(ctx, &pb.PublishRequest{
		Key:  key,
		Data: data,
	})
	return err
}

func (c *PubSubClient) Subscribe(ctx context.Context, key string, handler func(data string)) error {
	stream, err := c.client.Subscribe(ctx, &pb.SubscribeRequest{
		Key: key,
	})
	if err != nil {
		return err
	}

	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				log.Printf("Subscription ended: %v", err)
				return
			}
			handler(event.GetData())
		}
	}()

	return nil
}
