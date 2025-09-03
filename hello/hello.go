package hello

import (
	"context"
	"fmt"
	"time"
)

type HelloRequest struct {
	T time.Time `json:"time"`
}

func Hello(ctx context.Context, req HelloRequest) error {
	fmt.Println("Hello...")

	return nil
}
