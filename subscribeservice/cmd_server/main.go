package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	pb "subscribeservice/proto/pubsub"
	"syscall"
	_ "time"

	"google.golang.org/grpc"
	"subpub"
	"subscribeservice/service"
)

func main() {
	cfg, err := service.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := service.NewLogger()
	bus := subpub.NewSubPub()

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingUnaryInterceptor(logger)),
		grpc.StreamInterceptor(loggingStreamInterceptor(logger)),
	)

	pb.RegisterPubSubServer(grpcServer, service.NewPubSubService(bus, logger))

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		logger.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		logger.Printf("Starting gRPC server on %s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatalf("Failed to serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	grpcServer.GracefulStop()
	if err := bus.Close(ctx); err != nil {
		logger.Printf("Failed to close bus: %v", err)
	}
	logger.Println("Server stopped")
}

func loggingUnaryInterceptor(logger *log.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Printf("Unary call: %s", info.FullMethod)
		return handler(ctx, req)
	}
}

func loggingStreamInterceptor(logger *log.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		logger.Printf("Stream call: %s", info.FullMethod)
		return handler(srv, ss)
	}
}
