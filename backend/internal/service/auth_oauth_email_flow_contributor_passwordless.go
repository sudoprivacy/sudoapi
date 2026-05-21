// sudoapi: Google contributor OAuth passwordless signup.

package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

func (s *AuthService) RegisterVerifiedOAuthEmailContributorAccountWithRandomPassword(
	ctx context.Context,
	email string,
	signupSource string,
) (*TokenPair, *User, error) {
	if s == nil {
		return nil, nil, ErrServiceUnavailable
	}
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		return nil, nil, ErrRegDisabled
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(email) > 255 {
		return nil, nil, ErrEmailVerifyRequired
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, nil, ErrEmailVerifyRequired
	}
	if isReservedEmail(email) {
		return nil, nil, ErrEmailReserved
	}
	if err := s.validateRegistrationEmailPolicy(ctx, email); err != nil {
		return nil, nil, err
	}

	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, nil, ErrServiceUnavailable
	}
	if existsEmail {
		return nil, nil, ErrEmailExists
	}

	randomPassword, err := randomHexString(32)
	if err != nil {
		return nil, nil, ErrServiceUnavailable
	}
	hashedPassword, err := s.HashPassword(randomPassword)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	signupSource = normalizeOAuthSignupSource(signupSource)
	user := &User{
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         RoleAccountContributor,
		Balance:      0,
		Concurrency:  1,
		RPMLimit:     0,
		Status:       StatusActive,
		SignupSource: signupSource,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, ErrEmailExists) {
			return nil, nil, ErrEmailExists
		}
		return nil, nil, ErrServiceUnavailable
	}

	tokenPair, err := s.GenerateTokenPair(ctx, user, "")
	if err != nil {
		_ = s.RollbackOAuthEmailAccountCreation(ctx, user.ID, "")
		return nil, nil, fmt.Errorf("generate token pair: %w", err)
	}
	return tokenPair, user, nil
}
