package service

import (
	"context"
	"sync"
	"time"

	"github.com/nutochk/vk-test/pkg/subpub"
	"go.uber.org/zap"
)

type Service struct {
	pubsub subpub.SubPub
	logger *zap.Logger
	mu     sync.RWMutex
}

func New(logger *zap.Logger) *Service {
	return &Service{
		pubsub: subpub.NewSubPub(),
		logger: logger,
	}
}

func (s *Service) Subscribe(ctx context.Context, key string) (<-chan interface{}, error) {
	msgChan := make(chan interface{}, 100)

	handler := func(msg interface{}) { //???
		select {
		case msgChan <- msg:
		case <-ctx.Done():
			return
		}
	}

	sub, err := s.pubsub.Subscribe(key, handler)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(msgChan)
	}()

	return msgChan, nil
}

func (s *Service) Publish(ctx context.Context, key string, data interface{}) error {
	return s.pubsub.Publish(key, data)
}

func (s *Service) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.pubsub.Close(ctx)
}
