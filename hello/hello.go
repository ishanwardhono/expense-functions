package hello

import (
	"context"
	"time"
)

type HelloRequest struct {
	T time.Time `json:"time"`
}

func Hello(ctx context.Context, req HelloRequest) (string, error) {

	return "Test Restart Docker!", nil
}
