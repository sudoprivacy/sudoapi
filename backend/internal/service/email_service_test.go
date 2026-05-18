// sudoapi: SMTP STARTTLS auth support.

package service

import (
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSMTPAuthSelectsLoginWhenAdvertised(t *testing.T) {
	auth := SmartAuth("user@example.com", "password")
	proto, initialResp, err := auth.Start(&smtp.ServerInfo{
		Name: "smtp.example.com",
		TLS:  true,
		Auth: []string{"LOGIN"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "LOGIN", proto)
	assert.Nil(t, initialResp)

	username, err := auth.Next(nil, true)
	assert.NoError(t, err)
	assert.Equal(t, []byte("user@example.com"), username)

	password, err := auth.Next(nil, true)
	assert.NoError(t, err)
	assert.Equal(t, []byte("password"), password)
}

func TestSMTPAuthFallsBackToPlain(t *testing.T) {
	auth := SmartAuth("user@example.com", "password")
	proto, initialResp, err := auth.Start(&smtp.ServerInfo{
		Name: "smtp.example.com",
		TLS:  true,
		Auth: []string{"PLAIN"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "PLAIN", proto)
	assert.Equal(t, []byte("\x00user@example.com\x00password"), initialResp)
}
