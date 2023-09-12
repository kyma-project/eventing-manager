package main

import (
	"os"

	"github.com/kyma-project/eventing-manager/sink/internal/handler"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default port
	}

	sHandler := handler.NewSinkHandler(logger)
	err = sHandler.Start(port)
	if err != nil {
		logger.Error("failed to start SinkHandler", zap.Error(err))
	}
}
