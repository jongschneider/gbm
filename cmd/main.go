package main

import (
	"gbm/cmd/service"
	"os"
)

func main() {
	defer service.CloseLogFile()

	err := service.Execute()
	if err != nil {
		service.PrintError("%v\n", err)
		os.Exit(1)
	}
}
