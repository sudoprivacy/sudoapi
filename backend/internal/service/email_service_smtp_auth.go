// sudoapi: SMTP STARTTLS and LOGIN authentication support.

package service

import (
	"fmt"
	"net/smtp"
	"strings"
)

func useImplicitSMTPTLS(config *SMTPConfig) bool {
	return config.UseTLS && config.Port == 465
}

func requireStartTLS(config *SMTPConfig) bool {
	return config.UseTLS && !useImplicitSMTPTLS(config)
}

func newSMTPAuth(username, password, host string) smtp.Auth {
	return &smtpAuth{
		username: username,
		password: password,
		host:     host,
	}
}

type smtpAuth struct {
	username string
	password string
	host     string
	method   string
}

func (a *smtpAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && !isLocalSMTPServer(server.Name) {
		return "", nil, fmt.Errorf("unencrypted smtp auth is not allowed")
	}

	if smtpAuthSupports(server.Auth, "LOGIN") {
		a.method = "LOGIN"
		return "LOGIN", nil, nil
	}
	if smtpAuthSupports(server.Auth, "PLAIN") {
		a.method = "PLAIN"
		resp := "\x00" + a.username + "\x00" + a.password
		return "PLAIN", []byte(resp), nil
	}

	return "", nil, fmt.Errorf("server does not support PLAIN or LOGIN authentication")
}

func (a *smtpAuth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	if a.method != "LOGIN" {
		return nil, fmt.Errorf("unexpected smtp auth challenge")
	}
	if a.username != "" {
		username := a.username
		a.username = ""
		return []byte(username), nil
	}
	return []byte(a.password), nil
}

func smtpAuthSupports(auth []string, mechanism string) bool {
	for _, supported := range auth {
		if strings.EqualFold(supported, mechanism) {
			return true
		}
	}
	return false
}

func isLocalSMTPServer(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}
