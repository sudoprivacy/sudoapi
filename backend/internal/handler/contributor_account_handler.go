package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ContributorAccountHandler exposes the narrow account-contributor surface.
type ContributorAccountHandler struct {
	adminService       service.AdminService
	accountTestService *service.AccountTestService
}

func NewContributorAccountHandler(adminService service.AdminService, accountTestService *service.AccountTestService) *ContributorAccountHandler {
	return &ContributorAccountHandler{adminService: adminService, accountTestService: accountTestService}
}

func (h *ContributorAccountHandler) List(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found")
		return
	}
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 100 {
		search = search[:100]
	}
	accounts, total, err := h.adminService.ListContributorAccounts(
		c.Request.Context(),
		subject.UserID,
		page,
		pageSize,
		c.Query("platform"),
		c.Query("type"),
		c.Query("status"),
		search,
		c.DefaultQuery("sort_by", "created_at"),
		c.DefaultQuery("sort_order", "desc"),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]admin.AccountWithConcurrency, len(accounts))
	for i := range accounts {
		out[i] = admin.AccountWithConcurrency{Account: dto.AccountFromService(&accounts[i])}
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *ContributorAccountHandler) GetByID(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found")
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	account, err := h.adminService.GetContributorAccount(c.Request.Context(), subject.UserID, accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, admin.AccountWithConcurrency{Account: dto.AccountFromService(account)})
}

func (h *ContributorAccountHandler) Create(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found")
		return
	}
	var req admin.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	account, err := h.adminService.CreateContributorAccount(c.Request.Context(), subject.UserID, &service.CreateAccountInput{
		Name:               req.Name,
		Notes:              req.Notes,
		Platform:           req.Platform,
		Type:               req.Type,
		Credentials:        req.Credentials,
		Extra:              req.Extra,
		ProxyID:            req.ProxyID,
		Concurrency:        req.Concurrency,
		Priority:           req.Priority,
		RateMultiplier:     req.RateMultiplier,
		LoadFactor:         req.LoadFactor,
		ExpiresAt:          req.ExpiresAt,
		AutoPauseOnExpired: req.AutoPauseOnExpired,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, admin.AccountWithConcurrency{Account: dto.AccountFromService(account)})
}

func (h *ContributorAccountHandler) Update(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found")
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req admin.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	account, err := h.adminService.UpdateContributorAccount(c.Request.Context(), subject.UserID, accountID, &service.UpdateAccountInput{
		Name:               req.Name,
		Notes:              req.Notes,
		Type:               req.Type,
		Credentials:        req.Credentials,
		Extra:              req.Extra,
		ProxyID:            req.ProxyID,
		Concurrency:        req.Concurrency,
		Priority:           req.Priority,
		RateMultiplier:     req.RateMultiplier,
		LoadFactor:         req.LoadFactor,
		ExpiresAt:          req.ExpiresAt,
		AutoPauseOnExpired: req.AutoPauseOnExpired,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, admin.AccountWithConcurrency{Account: dto.AccountFromService(account)})
}

func (h *ContributorAccountHandler) Test(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found")
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if _, err := h.adminService.GetContributorAccount(c.Request.Context(), subject.UserID, accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var req admin.TestAccountRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.accountTestService.TestAccountConnection(c, accountID, req.ModelID, req.Prompt, req.Mode); err != nil {
		return
	}
}

func (h *ContributorAccountHandler) ListProxies(c *gin.Context) {
	proxies, err := h.adminService.GetAllProxies(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.Proxy, 0, len(proxies))
	for i := range proxies {
		out = append(out, *dto.ProxyFromService(&proxies[i]))
	}
	response.Success(c, out)
}
