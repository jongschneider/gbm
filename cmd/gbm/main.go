package main

import (
	"os"

	"gbm/cmd/gbm/service"
)

func main() {
	defer service.CloseLogFile()

	if err := service.Execute(); err != nil {
		service.PrintError(err)
		os.Exit(1)
	}
}
