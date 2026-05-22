// sudoapi: Account contributor review workflow.

package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ContributorAccountHandler exposes the narrow account-contributor surface.
type ContributorAccountHandler struct {
	adminService       service.AdminService
	accountTestService *service.AccountTestService
	oauthService       *service.OAuthService
	// sudoapi: Contributor account OpenAI OAuth self-service authorization.
	openaiOAuthService *service.OpenAIOAuthService
}

// sudoapi: Contributor account OpenAI OAuth self-service authorization.
func NewContributorAccountHandler(adminService service.AdminService, accountTestService *service.AccountTestService, oauthService *service.OAuthService, openaiOAuthService *service.OpenAIOAuthService) *ContributorAccountHandler {
	return &ContributorAccountHandler{
		adminService:       adminService,
		accountTestService: accountTestService,
		oauthService:       oauthService,
		openaiOAuthService: openaiOAuthService,
	}
}

func normalizeCountryParam(country string) string {
	return strings.ToUpper(strings.TrimSpace(country))
}

func countryFromRequest(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if country := normalizeCountryParam(c.Query("country")); country != "" {
		return country
	}
	return normalizeCountryParam(c.Query("country_code"))
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
		ContributorCountry: normalizeCountryParam(req.ContributorCountry),
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

// sudoapi: Account contributor review workflow.
func (h *ContributorAccountHandler) ListProxies(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found")
		return
	}
	proxy, err := h.adminService.SelectContributorProxy(c.Request.Context(), subject.UserID, countryFromRequest(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.Proxy, 0, 1)
	if proxy != nil {
		out = append(out, *dto.ProxyFromService(proxy))
	}
	response.Success(c, out)
}

func (h *ContributorAccountHandler) ReleaseProxyReservation(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found")
		return
	}
	if err := h.adminService.ReleaseContributorProxyReservations(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "proxy reservation released"})
}

func (h *ContributorAccountHandler) GenerateAuthURL(c *gin.Context) {
	var req admin.GenerateAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = admin.GenerateAuthURLRequest{}
	}

	result, err := h.oauthService.GenerateAuthURL(c.Request.Context(), req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

func (h *ContributorAccountHandler) GenerateSetupTokenURL(c *gin.Context) {
	var req admin.GenerateAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = admin.GenerateAuthURLRequest{}
	}

	result, err := h.oauthService.GenerateSetupTokenURL(c.Request.Context(), req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

func (h *ContributorAccountHandler) ExchangeCode(c *gin.Context) {
	var req admin.ExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tokenInfo, err := h.oauthService.ExchangeCode(c.Request.Context(), &service.ExchangeCodeInput{
		SessionID: req.SessionID,
		Code:      req.Code,
		ProxyID:   req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

func (h *ContributorAccountHandler) ExchangeSetupTokenCode(c *gin.Context) {
	var req admin.ExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tokenInfo, err := h.oauthService.ExchangeCode(c.Request.Context(), &service.ExchangeCodeInput{
		SessionID: req.SessionID,
		Code:      req.Code,
		ProxyID:   req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

// sudoapi: Contributor account OpenAI OAuth self-service authorization.
func (h *ContributorAccountHandler) RefreshOpenAIToken(c *gin.Context) {
	var req admin.OpenAIRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		refreshToken = strings.TrimSpace(req.RT)
	}
	if refreshToken == "" {
		response.BadRequest(c, "refresh_token is required")
		return
	}

	var proxyURL string
	if req.ProxyID != nil {
		proxy, err := h.adminService.GetProxy(c.Request.Context(), *req.ProxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		clientID, _ = openai.OAuthClientConfigByPlatform(service.PlatformOpenAI)
	}

	tokenInfo, err := h.openaiOAuthService.RefreshTokenWithClientID(c.Request.Context(), refreshToken, proxyURL, clientID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

// sudoapi: Contributor account OpenAI OAuth self-service authorization.
func (h *ContributorAccountHandler) GenerateOpenAIAuthURL(c *gin.Context) {
	var req admin.OpenAIGenerateAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = admin.OpenAIGenerateAuthURLRequest{}
	}

	result, err := h.openaiOAuthService.GenerateAuthURL(
		c.Request.Context(),
		req.ProxyID,
		req.RedirectURI,
		service.PlatformOpenAI,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// sudoapi: Contributor account OpenAI OAuth self-service authorization.
func (h *ContributorAccountHandler) ExchangeOpenAICode(c *gin.Context) {
	var req admin.OpenAIExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tokenInfo, err := h.openaiOAuthService.ExchangeCode(c.Request.Context(), &service.OpenAIExchangeCodeInput{
		SessionID:   req.SessionID,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: req.RedirectURI,
		ProxyID:     req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}
