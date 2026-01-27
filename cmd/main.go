package main

import (
	"gbm/cmd/service"
	"os"
)

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	defer service.CloseLogFile()

	err := service.Execute()
	if err != nil {
		service.PrintError("%v\n", err)
		return 1
	}
	return 0
}
