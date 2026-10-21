package main

import (
	"context"
	"os"

	"github.com/sadiksaifi/appsum/internal/app"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	return app.Run(context.Background())
}
