package main

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	_ "github.com/ishanwardhono/expense-function"
)

func main() {
	log.Println("Starting Functions Framework...")
	if err := funcframework.StartHostPort("localhost", os.Getenv("PORT")); err != nil {
		log.Fatalf("funcframework.StartHostPort: %v\n", err)
	}
}
