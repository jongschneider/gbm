package jira

import (
	"log"
	"os/exec"
)

type Service struct {
	User string
}

func NewService(user string) *Service {
	if _, err := exec.LookPath("jira"); err != nil {
		log.Fatal("jira command not found in PATH - please install jira CLI")
	}
	return &Service{User: user}
}
