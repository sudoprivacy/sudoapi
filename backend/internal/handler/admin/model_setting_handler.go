package admin

import (
	"mime/multipart"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelSettingHandler struct {
	modelSettingService *service.ModelSettingService
}

func NewModelSettingHandler(modelSettingService *service.ModelSettingService) *ModelSettingHandler {
	return &ModelSettingHandler{modelSettingService: modelSettingService}
}

func (h *ModelSettingHandler) Get(c *gin.Context) {
	if h == nil || h.modelSettingService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("MODEL_SETTING_NOT_READY", "model setting service not ready"))
		return
	}
	response.Success(c, h.modelSettingService.Status())
}

func (h *ModelSettingHandler) Upload(c *gin.Context) {
	if h == nil || h.modelSettingService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("MODEL_SETTING_NOT_READY", "model setting service not ready"))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MISSING_FILE", "csv file is required"))
		return
	}
	defer closeMultipartFile(file)

	status, err := h.modelSettingService.UploadCSV(header.Filename, file)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

func closeMultipartFile(file multipart.File) {
	if file != nil {
		_ = file.Close()
	}
}
