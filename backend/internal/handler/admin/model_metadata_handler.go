// sudoapi: Model market.

package admin

import (
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ModelMetadataHandler handles admin-maintained /models display metadata.
type ModelMetadataHandler struct {
	metadataService *service.ModelMetadataService
	modelSquareSvc  *service.ModelSquareService
}

func NewModelMetadataHandler(metadataService *service.ModelMetadataService, modelSquareSvc *service.ModelSquareService) *ModelMetadataHandler {
	return &ModelMetadataHandler{metadataService: metadataService, modelSquareSvc: modelSquareSvc}
}

type modelMetadataUpsertRequest struct {
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

type modelMetadataListItemResponse struct {
	ModelName     string                         `json:"model_name"`
	Platforms     []string                       `json:"platforms"`
	Metadata      modelMetadataViewResponse      `json:"metadata"`
	Override      *modelMetadataOverrideResponse `json:"override"`
	MissingFields []string                       `json:"missing_fields"`
}

type modelMetadataViewResponse struct {
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

type modelMetadataOverrideResponse struct {
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

// List GET /api/v1/admin/channels/model-metadata
func (h *ModelMetadataHandler) List(c *gin.Context) {
	items, err := h.metadataService.ListForAdmin(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]modelMetadataListItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, modelMetadataListItemToResponse(item))
	}
	response.Success(c, gin.H{"items": out})
}

// Upsert POST /api/v1/admin/channels/model-metadata
func (h *ModelMetadataHandler) Upsert(c *gin.Context) {
	var req modelMetadataUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.metadataService.Upsert(c.Request.Context(), &service.ModelMetadataOverride{
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
	h.modelSquareSvc.InvalidateAll()
	response.Success(c, modelMetadataOverrideToResponse(item))
}

// Delete DELETE /api/v1/admin/channels/model-metadata?model_name=...
func (h *ModelMetadataHandler) Delete(c *gin.Context) {
	modelName := strings.TrimSpace(c.Query("model_name"))
	if modelName == "" {
		response.ErrorFrom(c, service.ErrModelMetadataMissingModel)
		return
	}
	if err := h.metadataService.DeleteByModelName(c.Request.Context(), modelName); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.modelSquareSvc.InvalidateAll()
	response.Success(c, nil)
}

func modelMetadataListItemToResponse(item service.ModelMetadataListItem) modelMetadataListItemResponse {
	return modelMetadataListItemResponse{
		ModelName: item.ModelName,
		Platforms: item.Platforms,
		Metadata: modelMetadataViewResponse{
			DisplayName:      item.Metadata.DisplayName,
			Description:      item.Metadata.Description,
			Category:         item.Metadata.Category,
			ModelType:        item.Metadata.ModelType,
			ContextWindow:    item.Metadata.ContextWindow,
			MaxOutput:        item.Metadata.MaxOutput,
			Capabilities:     emptyStringSliceIfNil(item.Metadata.Capabilities),
			InputModalities:  emptyStringSliceIfNil(item.Metadata.InputModalities),
			OutputModalities: emptyStringSliceIfNil(item.Metadata.OutputModalities),
			SupportFlags:     emptyStringSliceIfNil(item.Metadata.SupportFlags),
			Featured:         item.Metadata.Featured,
			IconURL:          item.Metadata.IconURL,
		},
		Override:      modelMetadataOverrideToResponse(item.Override),
		MissingFields: emptyStringSliceIfNil(item.MissingFields),
	}
}

func modelMetadataOverrideToResponse(item *service.ModelMetadataOverride) *modelMetadataOverrideResponse {
	if item == nil {
		return nil
	}
	return &modelMetadataOverrideResponse{
		ID:               item.ID,
		ModelName:        item.ModelName,
		DisplayName:      item.DisplayName,
		Description:      item.Description,
		Category:         item.Category,
		ModelType:        item.ModelType,
		ContextWindow:    item.ContextWindow,
		MaxOutput:        item.MaxOutput,
		Capabilities:     emptyStringSliceIfNil(item.Capabilities),
		InputModalities:  emptyStringSliceIfNil(item.InputModalities),
		OutputModalities: emptyStringSliceIfNil(item.OutputModalities),
		SupportFlags:     emptyStringSliceIfNil(item.SupportFlags),
		Featured:         item.Featured,
		IconURL:          item.IconURL,
		CreatedAt:        formatMetadataTime(item.CreatedAt),
		UpdatedAt:        formatMetadataTime(item.UpdatedAt),
	}
}

func emptyStringSliceIfNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func formatMetadataTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
