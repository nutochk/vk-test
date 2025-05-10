package server

import (
	"context"
	"sync"

	"github.com/nutochkk/pubsub-service/internal/service"
	pubsub "github.com/nutochkk/pubsub-service/pkg/api"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PubSubServer struct {
	pubsub.UnimplementedPubSubServer
	service *service.Service
	logger  *zap.Logger
	wg      sync.WaitGroup
}

func New(service *service.Service, logger *zap.Logger) *PubSubServer {
	return &PubSubServer{
		service: service,
		logger:  logger,
	}
}

func (s *PubSubServer) Subscribe(req *pubsub.SubscribeRequest, stream pubsub.PubSub_SubscribeServer) error {
	ctx := stream.Context()
	s.wg.Add(1)
	defer s.wg.Done()

	msgChan, err := s.service.Subscribe(ctx, req.Key)
	if err != nil {
		s.logger.Error("subscribe failed", zap.Error(err))
		return status.Errorf(codes.Internal, "subscribe failed")
	}

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				return nil
			}

			data, ok := msg.(string)
			if !ok {
				s.logger.Warn("invalid message type", zap.Any("msg", msg))
				continue
			}

			if err := stream.Send(&pubsub.Event{Data: data}); err != nil {
				s.logger.Error("stream send failed", zap.Error(err))
				return status.Errorf(codes.Internal, "stream send failed")
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (s *PubSubServer) Publish(ctx context.Context, req *pubsub.PublishRequest) (*emptypb.Empty, error) {
	if err := s.service.Publish(ctx, req.Key, req.Data); err != nil {
		s.logger.Error("publish failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "publish failed")
	}
	return &emptypb.Empty{}, nil
}

func (s *PubSubServer) GracefulStop() {
	s.wg.Wait()
}
