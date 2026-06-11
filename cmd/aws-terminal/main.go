package main

import (
	"log"

	"aws-terminal/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
