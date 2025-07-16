package main

import (
	"log"

	// Blank-import the function package so the init() runs
	_ "github.com/ishanwardhono/gcp-functions/hello"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func main() {
	log.Println("Starting Functions Framework...")
	if err := funcframework.StartHostPort("localhost", "8199"); err != nil {
		log.Fatalf("funcframework.StartHostPort: %v\n", err)
	}
}
