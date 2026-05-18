// sudoapi: SMTP STARTTLS auth support.

package service

import (
	"errors"
	"net/smtp"
	"strings"
)

func SmartAuth(username, password string) smtp.Auth {
	return &smartAuth{
		username: username,
		password: password,
	}
}

// smartAuth try LOGIN auth and PLAIN auth
type smartAuth struct {
	username string
	password string
	method   string
	step     int
}

func (a *smartAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if smtpAuthSupports(server.Auth, "LOGIN") {
		a.method = "LOGIN"
		return "LOGIN", nil, nil
	}
	if smtpAuthSupports(server.Auth, "PLAIN") {
		a.method = "PLAIN"
		resp := "\x00" + a.username + "\x00" + a.password
		return "PLAIN", []byte(resp), nil
	}

	return "", nil, errors.New("server does not support PLAIN or LOGIN authentication")
}

func (a *smartAuth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	if a.method != "LOGIN" {
		return nil, errors.New("unexpected smtp auth challenge")
	}
	switch a.step {
	case 0:
		a.step++
		return []byte(a.username), nil
	case 1:
		a.step++
		return []byte(a.password), nil
	default:
		return nil, errors.New("unexpected server response during LOGIN auth")
	}
}

func smtpAuthSupports(auth []string, mechanism string) bool {
	for _, supported := range auth {
		if strings.EqualFold(supported, mechanism) {
			return true
		}
	}
	return false
}
