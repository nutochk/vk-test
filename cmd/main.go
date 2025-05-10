package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nutochkk/pubsub-service/internal/config"
	"github.com/nutochkk/pubsub-service/internal/server"
	"github.com/nutochkk/pubsub-service/internal/service"
	pubsub "github.com/nutochkk/pubsub-service/pkg/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.New("config/config.yaml")
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	svc := service.New(logger)
	grpcServer := grpc.NewServer()
	pubsubServer := server.New(svc, logger)

	pubsub.RegisterPubSubServer(grpcServer, pubsubServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		logger.Info("starting gRPC server", zap.Int("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down server...")
	_, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	grpcServer.GracefulStop()
	pubsubServer.GracefulStop()

	if err := svc.Close(); err != nil {
		logger.Error("service close failed", zap.Error(err))
	}

	logger.Info("server stopped")
}
