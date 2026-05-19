package main

import (
	"fmt"
	"os"

	"dualgit/internal/app"
)

func main() {
	if err := app.New().Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
