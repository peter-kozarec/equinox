package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
	"syscall"
)

const version = "v0.1.0"

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}(logger)

	logger.Info(fmt.Sprintf("mrx %s", version))
	defer logger.Info("done")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	router := bus.NewRouter(logger, 100)
	router.TickHandler = func(*model.Tick) error {
		return nil
	}
	go router.Exec(ctx, func(_ context.Context) error {
		_ = router.Post(bus.TickEvent, &model.Tick{
			TimeStamp: 0,
			Ask:       0,
			Bid:       0,
			AskVolume: 0,
			BidVolume: 0,
		})
		return nil
	})

	<-router.Done()
	router.PrintStatistics()
}
