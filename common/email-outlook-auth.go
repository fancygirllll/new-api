package common

import (
	"errors"
	"net/smtp"
	"strings"
)

type outlookAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &outlookAuth{username, password}
}

func (a *outlookAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *outlookAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown fromServer")
		}
	}
	return nil, nil
}

func isOutlookServer(server string) bool {
	// Support Outlook and OFB mailboxes across multiple regions
	// Ideally an option should be added to choose whether to log in via LOGIN
	// Temporary compatibility workaround for now
	return strings.Contains(server, "outlook") || strings.Contains(server, "onmicrosoft")
}
