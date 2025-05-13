package main

import (
	"fmt"
	"go.uber.org/zap"
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

}
