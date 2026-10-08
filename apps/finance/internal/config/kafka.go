package config

import (
	"context"
	"fmt"

	"github.com/kilip/omed/finance/internal/kafka"
	"github.com/kilip/omed/finance/internal/repository"
	"github.com/kilip/omed/finance/internal/service"
)

// StartEventConsumer jalan di background sampai ctx dibatalkan.
func StartEventConsumer(ctx context.Context, state State) error {
	users := repository.NewUserRepository(state.EntClient, state.Log)
	works := repository.NewWorkspaceRepository(state.EntClient, state.Log)
	sync := service.NewEventSync(users, works)

	c, err := kafka.NewConsumer(
		state.Config.KafkaBrokers,
		"omed-finance",
		map[string]kafka.Handler{
			service.TopicUser: sync.HandleUser,
			service.TopicTeam: sync.HandleTeam,
		},
		state.Log,
	)
	if err != nil {
		return fmt.Errorf("init kafka consumer: %w", err)
	}

	go c.Run(ctx)
	return nil
}
