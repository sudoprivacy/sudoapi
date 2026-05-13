// sudoapi: CSV-style admin batch user creation.

package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// BatchCreateUserRow represents a single user in a batch-create payload.
// Validation is intentionally lenient here; the service layer reports per-row
// errors so the client can surface them instead of failing the whole request.
type BatchCreateUserRow struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	Username    string  `json:"username"`
	Balance     float64 `json:"balance"`
	Concurrency int     `json:"concurrency"`
	RPMLimit    int     `json:"rpm_limit"`
}

// BatchCreateUsersRequest is the request body for POST /admin/users/batch.
type BatchCreateUsersRequest struct {
	Users       []BatchCreateUserRow `json:"users" binding:"required,min=1"`
	SkipOnError bool                 `json:"skip_on_error"`
}

// BatchCreateUserResultDTO mirrors service.BatchCreateUserResult for the wire.
type BatchCreateUserResultDTO struct {
	Index     int    `json:"index"`
	Email     string `json:"email"`
	Success   bool   `json:"success"`
	UserID    int64  `json:"user_id,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

// BatchCreateUsersResponse mirrors service.BatchCreateUsersOutput for the wire.
type BatchCreateUsersResponse struct {
	Total   int                        `json:"total"`
	Created int                        `json:"created"`
	Failed  int                        `json:"failed"`
	Aborted bool                       `json:"aborted"`
	Results []BatchCreateUserResultDTO `json:"results"`
}

// BatchCreate handles bulk user creation from a CSV-style payload.
// POST /api/v1/admin/users/batch
func (h *UserHandler) BatchCreate(c *gin.Context) {
	var req BatchCreateUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	rows := make([]service.CreateUserInput, len(req.Users))
	for i, u := range req.Users {
		rows[i] = service.CreateUserInput{
			Email:       u.Email,
			Password:    u.Password,
			Username:    u.Username,
			Balance:     u.Balance,
			Concurrency: u.Concurrency,
			RPMLimit:    u.RPMLimit,
		}
	}

	out, err := h.adminService.BatchCreateUsers(c.Request.Context(), &service.BatchCreateUsersInput{
		Rows:        rows,
		SkipOnError: req.SkipOnError,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	resp := BatchCreateUsersResponse{
		Total:   out.Total,
		Created: out.Created,
		Failed:  out.Failed,
		Aborted: out.Aborted,
		Results: make([]BatchCreateUserResultDTO, len(out.Results)),
	}
	for i, r := range out.Results {
		resp.Results[i] = BatchCreateUserResultDTO{
			Index:     r.Index,
			Email:     r.Email,
			Success:   r.Success,
			UserID:    r.UserID,
			ErrorCode: r.ErrorCode,
			ErrorMsg:  r.ErrorMsg,
		}
	}
	response.Success(c, resp)
}
