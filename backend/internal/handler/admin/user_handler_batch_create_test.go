// sudoapi: CSV-style admin batch user creation.

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *stubAdminService) BatchCreateUsers(ctx context.Context, input *service.BatchCreateUsersInput) (*service.BatchCreateUsersOutput, error) {
	out := &service.BatchCreateUsersOutput{Total: len(input.Rows), Results: make([]service.BatchCreateUserResult, 0, len(input.Rows))}
	for i, row := range input.Rows {
		out.Results = append(out.Results, service.BatchCreateUserResult{Index: i + 1, Email: row.Email, Success: true, UserID: int64(1000 + i)})
		out.Created++
	}
	return out, nil
}

func TestUserHandler_BatchCreate_HappyPath(t *testing.T) {
	router, _ := setupAdminRouter()
	userHandler := NewUserHandler(newStubAdminService(), nil, nil, nil)
	router.POST("/api/v1/admin/users/batch", userHandler.BatchCreate)

	body, _ := json.Marshal(map[string]any{
		"users": []map[string]any{
			{"email": "a@test.com", "password": "passwd1"},
			{"email": "b@test.com", "password": "passwd2"},
		},
		"skip_on_error": true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Total   int  `json:"total"`
			Created int  `json:"created"`
			Failed  int  `json:"failed"`
			Aborted bool `json:"aborted"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 2, resp.Data.Total)
	require.Equal(t, 2, resp.Data.Created)
	require.Equal(t, 0, resp.Data.Failed)
}

func TestUserHandler_BatchCreate_EmptyUsersRejected(t *testing.T) {
	router, _ := setupAdminRouter()
	userHandler := NewUserHandler(newStubAdminService(), nil, nil, nil)
	router.POST("/api/v1/admin/users/batch", userHandler.BatchCreate)

	body, _ := json.Marshal(map[string]any{"users": []any{}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
