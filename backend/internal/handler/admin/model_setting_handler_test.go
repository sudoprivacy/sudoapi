//go:build unit

package admin

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelSettingHandlerUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svcCfg := &config.Config{}
	svcCfg.Pricing.DataDir = t.TempDir()
	svc := service.NewModelSettingService(svcCfg)
	h := NewModelSettingHandler(svc)

	body, contentType := modelSettingMultipartBody(t, "models.csv", "serial_number,id\n1,gpt-a\n")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/model_setting", body)
	c.Request.Header.Set("Content-Type", contentType)

	h.Upload(c)

	require.Equal(t, http.StatusOK, w.Code)
	serial, ok := svc.LookupModelSerialNumber("gpt-a")
	require.True(t, ok)
	require.Equal(t, 1, serial)
}

func TestModelSettingHandlerUploadRejectsInvalidCSV(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svcCfg := &config.Config{}
	svcCfg.Pricing.DataDir = t.TempDir()
	svc := service.NewModelSettingService(svcCfg)
	h := NewModelSettingHandler(svc)

	body, contentType := modelSettingMultipartBody(t, "models.csv", "serial_number,id\nbad,gpt-a\n")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/model_setting", body)
	c.Request.Header.Set("Content-Type", contentType)

	h.Upload(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	_, ok := svc.LookupModelSerialNumber("gpt-a")
	require.False(t, ok)
}

func modelSettingMultipartBody(t *testing.T, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}
