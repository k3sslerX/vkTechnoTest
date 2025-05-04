package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subscribeservice/client"
)

func main() {
	cli, err := client.NewClient("localhost:1313", 5)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		_ = cli.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = cli.Subscribe(ctx, "news", func(data string) {
		log.Printf("Received news: %s", data)
	})
	if err != nil {
		log.Fatalf("Subscribe failed: %v", err)
	}

	go func() {
		for i := 0; i < 5; i++ {
			if err := cli.Publish(context.Background(), "news", fmt.Sprintf("News item %d", i)); err != nil {
				log.Printf("Publish failed: %v", err)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down client...")
}
