package main

import (
	"github.com/kyma-project/eventing-manager/sink/internal/handler"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sHandler := handler.NewSinkHandler(logger)
	err = sHandler.Start()
	if err != nil {
		logger.Error("failed to start SinkHandler", zap.Error(err))
	}
}
