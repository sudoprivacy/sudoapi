// sudoapi: Model catalog.

package admin

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service_model_catalog"
)

func NewModelCatalogHandler(
	catalogService *service_model_catalog.ModelCatalogService,
	metadataService *service_model_catalog.MetadataService,
	endpointConfigService *service_model_catalog.EndpointConfigService,
) *ModelCatalogHandler {
	return &ModelCatalogHandler{
		metadataService:       metadataService,
		catalogService:        catalogService,
		endpointConfigService: endpointConfigService,
	}
}

// ModelCatalogHandler handles admin-maintained /models display metadata.
type ModelCatalogHandler struct {
	catalogService        *service_model_catalog.ModelCatalogService
	metadataService       *service_model_catalog.MetadataService
	endpointConfigService *service_model_catalog.EndpointConfigService
}

// GetEndpointConfig handles getting global model catalog endpoint config.
// GET /api/v1/admin/model-catalog/endpoint-config
func (h *ModelCatalogHandler) GetEndpointConfig(c *gin.Context) {
	cfg, err := h.endpointConfigService.GetEndpointConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

// UpdateEndpointConfig handles replacing global model catalog endpoint config.
// PUT /api/v1/admin/model-catalog/endpoint-config
func (h *ModelCatalogHandler) UpdateEndpointConfig(c *gin.Context) {
	var req service_model_catalog.EndpointConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	cfg, err := h.endpointConfigService.SetEndpointConfig(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.catalogService.InvalidateAll()
	response.Success(c, cfg)
}

// ListMetadata GET /api/v1/admin/model-catalog/metadata
func (h *ModelCatalogHandler) ListMetadata(c *gin.Context) {
	items, err := h.metadataService.ListForAdmin(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rst := lo.Map(items, func(item service_model_catalog.MetadataListItem, index int) MetadataListItemDTO {
		return MetadataListItemDTO{
			ModelName: item.ModelName,
			Platforms: item.Platforms,
			Metadata: MetadataViewDTO{
				DisplayName:      item.Metadata.DisplayName,
				Description:      item.Metadata.Description,
				Category:         item.Metadata.Category,
				ModelType:        item.Metadata.ModelType,
				ContextWindow:    item.Metadata.ContextWindow,
				MaxOutput:        item.Metadata.MaxOutput,
				Capabilities:     item.Metadata.Capabilities,
				InputModalities:  item.Metadata.InputModalities,
				OutputModalities: item.Metadata.OutputModalities,
				SupportFlags:     item.Metadata.SupportFlags,
				Featured:         item.Metadata.Featured,
				IconURL:          item.Metadata.IconURL,
			},
			Override:      MetadataOverrideToDTO(item.Override),
			MissingFields: item.MissingFields,
		}
	})
	response.Success(c, gin.H{"items": rst})
}

// UpsertMetadata POST /api/v1/admin/model-catalog/metadata
func (h *ModelCatalogHandler) UpsertMetadata(c *gin.Context) {
	var req MetadataUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.metadataService.Upsert(c.Request.Context(), &service_model_catalog.MetadataOverride{
		ModelName:        req.ModelName,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		Category:         req.Category,
		ModelType:        req.ModelType,
		ContextWindow:    req.ContextWindow,
		MaxOutput:        req.MaxOutput,
		Capabilities:     req.Capabilities,
		InputModalities:  req.InputModalities,
		OutputModalities: req.OutputModalities,
		SupportFlags:     req.SupportFlags,
		Featured:         req.Featured,
		IconURL:          req.IconURL,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.catalogService.InvalidateAll()
	response.Success(c, MetadataOverrideToDTO(item))
}

// DeleteMetadata DELETE /api/v1/admin/model-catalog/metadata?model_name=...
func (h *ModelCatalogHandler) DeleteMetadata(c *gin.Context) {
	modelName := strings.TrimSpace(c.Query("model_name"))
	if modelName == "" {
		response.ErrorFrom(c, service_model_catalog.ErrMetadataMissingModel)
		return
	}
	if err := h.metadataService.DeleteByModelName(c.Request.Context(), modelName); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.catalogService.InvalidateAll()
	response.Success(c, nil)
}

type (
	MetadataUpsertRequest struct {
		ModelName        string   `json:"model_name" binding:"required,max=200"`
		DisplayName      string   `json:"display_name" binding:"max=200"`
		Description      string   `json:"description"`
		Category         string   `json:"category" binding:"omitempty,max=50"`
		ModelType        string   `json:"model_type"`
		ContextWindow    int      `json:"context_window" binding:"omitempty,min=0"`
		MaxOutput        int      `json:"max_output" binding:"omitempty,min=0"`
		Capabilities     []string `json:"capabilities"`
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
		SupportFlags     []string `json:"support_flags"`
		Featured         bool     `json:"featured"`
		IconURL          string   `json:"icon_url"`
	}

	MetadataListItemDTO struct {
		ModelName     string               `json:"model_name"`
		Platforms     []string             `json:"platforms"`
		Metadata      MetadataViewDTO      `json:"metadata"`
		Override      *MetadataOverrideDTO `json:"override"`
		MissingFields []string             `json:"missing_fields"`
	}

	MetadataViewDTO struct {
		DisplayName      string   `json:"display_name"`
		Description      string   `json:"description"`
		Category         string   `json:"category"`
		ModelType        string   `json:"model_type"`
		ContextWindow    int      `json:"context_window"`
		MaxOutput        int      `json:"max_output"`
		Capabilities     []string `json:"capabilities"`
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
		SupportFlags     []string `json:"support_flags"`
		Featured         bool     `json:"featured"`
		IconURL          string   `json:"icon_url"`
	}

	MetadataOverrideDTO struct {
		ID               int64    `json:"id"`
		ModelName        string   `json:"model_name"`
		DisplayName      string   `json:"display_name"`
		Description      string   `json:"description"`
		Category         string   `json:"category"`
		ModelType        string   `json:"model_type"`
		ContextWindow    int      `json:"context_window"`
		MaxOutput        int      `json:"max_output"`
		Capabilities     []string `json:"capabilities"`
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
		SupportFlags     []string `json:"support_flags"`
		Featured         bool     `json:"featured"`
		IconURL          string   `json:"icon_url"`
		CreatedAt        string   `json:"created_at"`
		UpdatedAt        string   `json:"updated_at"`
	}
)

func MetadataOverrideToDTO(item *service_model_catalog.MetadataOverride) *MetadataOverrideDTO {
	if item == nil {
		return nil
	}
	return &MetadataOverrideDTO{
		ID:               item.ID,
		ModelName:        item.ModelName,
		DisplayName:      item.DisplayName,
		Description:      item.Description,
		Category:         item.Category,
		ModelType:        item.ModelType,
		ContextWindow:    item.ContextWindow,
		MaxOutput:        item.MaxOutput,
		Capabilities:     item.Capabilities,
		InputModalities:  item.InputModalities,
		OutputModalities: item.OutputModalities,
		SupportFlags:     item.SupportFlags,
		Featured:         item.Featured,
		IconURL:          item.IconURL,
		CreatedAt:        formatMetadataTime(item.CreatedAt),
		UpdatedAt:        formatMetadataTime(item.UpdatedAt),
	}
}

func formatMetadataTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
