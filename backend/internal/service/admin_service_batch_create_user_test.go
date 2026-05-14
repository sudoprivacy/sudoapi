//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminService_BatchCreateUsers_AllSuccess(t *testing.T) {
	repo := &userRepoStub{nextID: 1}
	svc := &adminServiceImpl{userRepo: repo}

	out, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{
		Rows: []CreateUserInput{
			{Email: "a@test.com", Password: "passwd1"},
			{Email: "b@test.com", Password: "passwd2"},
		},
		SkipOnError: true,
	})
	require.NoError(t, err)
	require.Equal(t, 2, out.Total)
	require.Equal(t, 2, out.Created)
	require.Equal(t, 0, out.Failed)
	require.False(t, out.Aborted)
	require.True(t, out.Results[0].Success)
	require.True(t, out.Results[1].Success)
	require.Equal(t, "a@test.com", out.Results[0].Email)
}

func TestAdminService_BatchCreateUsers_InvalidEmailSkipsRow(t *testing.T) {
	repo := &userRepoStub{nextID: 1}
	svc := &adminServiceImpl{userRepo: repo}

	out, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{
		Rows: []CreateUserInput{
			{Email: "notanemail", Password: "passwd1"},
			{Email: "b@test.com", Password: "passwd2"},
		},
		SkipOnError: true,
	})
	require.NoError(t, err)
	require.Equal(t, 1, out.Created)
	require.Equal(t, 1, out.Failed)
	require.False(t, out.Results[0].Success)
	require.Equal(t, "INVALID_EMAIL", out.Results[0].ErrorCode)
	require.True(t, out.Results[1].Success)
}

func TestAdminService_BatchCreateUsers_WeakPassword(t *testing.T) {
	repo := &userRepoStub{nextID: 1}
	svc := &adminServiceImpl{userRepo: repo}

	out, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{
		Rows: []CreateUserInput{
			{Email: "a@test.com", Password: "short"},
		},
		SkipOnError: true,
	})
	require.NoError(t, err)
	require.Equal(t, 0, out.Created)
	require.Equal(t, 1, out.Failed)
	require.Equal(t, "WEAK_PASSWORD", out.Results[0].ErrorCode)
}

func TestAdminService_BatchCreateUsers_InPayloadDuplicate(t *testing.T) {
	repo := &userRepoStub{nextID: 1}
	svc := &adminServiceImpl{userRepo: repo}

	out, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{
		Rows: []CreateUserInput{
			{Email: "dup@test.com", Password: "passwd1"},
			{Email: "DUP@test.com", Password: "passwd2"}, // case-insensitive duplicate
			{Email: "ok@test.com", Password: "passwd3"},
		},
		SkipOnError: true,
	})
	require.NoError(t, err)
	require.Equal(t, 2, out.Created)
	require.Equal(t, 1, out.Failed)
	require.Equal(t, "DUPLICATE_IN_PAYLOAD", out.Results[1].ErrorCode)
}

func TestAdminService_BatchCreateUsers_DBEmailExists(t *testing.T) {
	// Force every Create to return ErrEmailExists.
	repo := &userRepoStub{createErr: ErrEmailExists}
	svc := &adminServiceImpl{userRepo: repo}

	out, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{
		Rows: []CreateUserInput{
			{Email: "a@test.com", Password: "passwd1"},
		},
		SkipOnError: true,
	})
	require.NoError(t, err)
	require.Equal(t, 1, out.Failed)
	require.Equal(t, "EMAIL_EXISTS", out.Results[0].ErrorCode)
}

func TestAdminService_BatchCreateUsers_AbortOnError(t *testing.T) {
	repo := &userRepoStub{nextID: 1}
	svc := &adminServiceImpl{userRepo: repo}

	out, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{
		Rows: []CreateUserInput{
			{Email: "a@test.com", Password: "passwd1"},
			{Email: "bad", Password: "passwd2"},
			{Email: "c@test.com", Password: "passwd3"},
		},
		SkipOnError: false,
	})
	require.NoError(t, err)
	require.True(t, out.Aborted)
	require.Equal(t, 1, out.Created)
	require.Equal(t, 1, out.Failed)
	require.Len(t, out.Results, 2, "third row should not be processed after abort")
}

func TestAdminService_BatchCreateUsers_EmptyRowsRejected(t *testing.T) {
	svc := &adminServiceImpl{userRepo: &userRepoStub{}}

	_, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{Rows: nil})
	require.Error(t, err)
}

func TestAdminService_BatchCreateUsers_ExceedsCap(t *testing.T) {
	svc := &adminServiceImpl{userRepo: &userRepoStub{nextID: 1}}

	rows := make([]CreateUserInput, BatchCreateUsersMaxRows+1)
	for i := range rows {
		rows[i] = CreateUserInput{Email: "u@test.com", Password: "passwd1"}
	}
	_, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{Rows: rows})
	require.Error(t, err)
}

func TestAdminService_BatchCreateUsers_NegativeBalance(t *testing.T) {
	svc := &adminServiceImpl{userRepo: &userRepoStub{nextID: 1}}

	out, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{
		Rows: []CreateUserInput{
			{Email: "a@test.com", Password: "passwd1", Balance: -1},
		},
		SkipOnError: true,
	})
	require.NoError(t, err)
	require.Equal(t, "INVALID_BALANCE", out.Results[0].ErrorCode)
}

func TestAdminService_BatchCreateUsers_NormalizesEmail(t *testing.T) {
	repo := &userRepoStub{nextID: 1}
	svc := &adminServiceImpl{userRepo: repo}

	out, err := svc.BatchCreateUsers(context.Background(), &BatchCreateUsersInput{
		Rows: []CreateUserInput{
			{Email: "  Mixed@Case.com  ", Password: "passwd1"},
		},
		SkipOnError: true,
	})
	require.NoError(t, err)
	require.Equal(t, 1, out.Created)
	require.Len(t, repo.created, 1)
	require.Equal(t, "mixed@case.com", repo.created[0].Email)
}
