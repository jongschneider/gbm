package main

import (
	"gbm/cmd/service"
	"os"
)

func main() {
	defer service.CloseLogFile()

	if err := service.Execute(); err != nil {
		service.PrintError("%v\n", err)
		os.Exit(1)
	}
}
