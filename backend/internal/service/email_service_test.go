package service

import (
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSMTPTLSMode(t *testing.T) {
	tests := []struct {
		name            string
		config          *SMTPConfig
		wantImplicitTLS bool
		wantRequireTLS  bool
	}{
		{
			name: "outlook 587 uses required starttls",
			config: &SMTPConfig{
				Host:   "smtp.office365.com",
				Port:   587,
				UseTLS: true,
			},
			wantRequireTLS: true,
		},
		{
			name: "smtps 465 uses implicit tls",
			config: &SMTPConfig{
				Host:   "smtp.example.com",
				Port:   465,
				UseTLS: true,
			},
			wantImplicitTLS: true,
		},
		{
			name: "tls disabled allows opportunistic starttls",
			config: &SMTPConfig{
				Host: "smtp.example.com",
				Port: 587,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := useImplicitSMTPTLS(tt.config); got != tt.wantImplicitTLS {
				t.Fatalf("useImplicitSMTPTLS() = %v, want %v", got, tt.wantImplicitTLS)
			}
			if got := requireStartTLS(tt.config); got != tt.wantRequireTLS {
				t.Fatalf("requireStartTLS() = %v, want %v", got, tt.wantRequireTLS)
			}
		})
	}
}

func TestSMTPAuthSelectsLoginWhenAdvertised(t *testing.T) {
	auth := newSMTPAuth("user@example.com", "password", "smtp.example.com")
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
	auth := newSMTPAuth("user@example.com", "password", "smtp.example.com")
	proto, initialResp, err := auth.Start(&smtp.ServerInfo{
		Name: "smtp.example.com",
		TLS:  true,
		Auth: []string{"PLAIN"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "PLAIN", proto)
	assert.Equal(t, []byte("\x00user@example.com\x00password"), initialResp)
}
