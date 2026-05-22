//go:build unit

package service

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParseModelSettingCSVValidAndDuplicate(t *testing.T) {
	records, summary, err := parseModelSettingCSV(bytes.NewBufferString("serial_number,owned_by,id\n2,OpenAI,gpt-b\n1,OpenAI,gpt-a\n3,OpenAI,gpt-a\n , , \n"))
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, 1, records["gpt-a"].SerialNumber)
	require.Equal(t, 2, records["gpt-b"].SerialNumber)
	require.Equal(t, 3, summary.TotalRows)
	require.Equal(t, 2, summary.LoadedRows)
	require.Equal(t, 1, summary.DuplicateRows)
	require.Equal(t, 1, summary.SkippedRows)
}

func TestParseModelSettingCSVRequiresColumns(t *testing.T) {
	_, _, err := parseModelSettingCSV(bytes.NewBufferString("serial_number,name\n1,gpt-a\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "id column")
}

func TestParseModelSettingCSVRejectsInvalidSerialNumber(t *testing.T) {
	_, _, err := parseModelSettingCSV(bytes.NewBufferString("serial_number,id\nabc,gpt-a\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid serial_number")
}

func TestParseModelSettingCSVRejectsEmptyID(t *testing.T) {
	_, _, err := parseModelSettingCSV(bytes.NewBufferString("serial_number,id\n1, \n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "model id cannot be empty")
}

func TestModelSettingUploadCSVReplacesFileAndState(t *testing.T) {
	dir := t.TempDir()
	svc := NewModelSettingService(&config.Config{})
	svc.cfg.Pricing.DataDir = dir

	status, err := svc.UploadCSV("models.csv", bytes.NewBufferString("serial_number,id\n1,gpt-a\n2,gpt-b\n"))
	require.NoError(t, err)
	require.Equal(t, 2, status.ModelCount)
	require.Equal(t, "uploaded", status.Source)
	require.FileExists(t, filepath.Join(dir, modelSettingDirName, modelSettingFileName))

	serial, ok := svc.LookupModelSerialNumber("GPT-A")
	require.True(t, ok)
	require.Equal(t, 1, serial)
}

func TestModelSettingUploadCSVDoesNotReplaceOnInvalidCSV(t *testing.T) {
	dir := t.TempDir()
	svc := NewModelSettingService(&config.Config{})
	svc.cfg.Pricing.DataDir = dir

	_, err := svc.UploadCSV("models.csv", bytes.NewBufferString("serial_number,id\n1,gpt-a\n"))
	require.NoError(t, err)
	target := filepath.Join(dir, modelSettingDirName, modelSettingFileName)
	before, err := os.ReadFile(target)
	require.NoError(t, err)

	_, err = svc.UploadCSV("models.csv", bytes.NewBufferString("serial_number,id\nbad,gpt-b\n"))
	require.Error(t, err)
	after, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))

	_, ok := svc.LookupModelSerialNumber("gpt-b")
	require.False(t, ok)
}
