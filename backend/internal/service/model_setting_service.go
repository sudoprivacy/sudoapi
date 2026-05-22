package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	modelSettingDirName  = "model_setting"
	modelSettingFileName = "models_grouped_id_desc.csv"
)

type ModelSettingRecord struct {
	SerialNumber int    `json:"serial_number"`
	ID           string `json:"id"`
}

type ModelSettingSummary struct {
	TotalRows      int `json:"total_rows"`
	LoadedRows     int `json:"loaded_rows"`
	DuplicateRows  int `json:"duplicate_rows"`
	SkippedRows    int `json:"skipped_rows"`
	HeaderRowCount int `json:"header_row_count"`
}

type ModelSettingStatus struct {
	FilePath   string              `json:"file_path"`
	FileName   string              `json:"file_name"`
	Source     string              `json:"source"`
	ModelCount int                 `json:"model_count"`
	UpdatedAt  string              `json:"updated_at"`
	Summary    ModelSettingSummary `json:"summary"`
}

type modelSettingState struct {
	recordsByID map[string]ModelSettingRecord
	summary     ModelSettingSummary
	sourcePath  string
	source      string
	updatedAt   time.Time
}

type ModelSettingService struct {
	cfg *config.Config

	mu    sync.RWMutex
	state modelSettingState
}

func NewModelSettingService(cfg *config.Config) *ModelSettingService {
	s := &ModelSettingService{cfg: cfg}
	if err := s.Initialize(); err != nil {
		logger.LegacyPrintf("service.model_setting", "[ModelSetting] Initial load failed: %v", err)
	}
	return s
}

func (s *ModelSettingService) Initialize() error {
	for _, candidate := range s.loadCandidates() {
		body, err := os.ReadFile(candidate.path)
		if err != nil {
			continue
		}
		records, summary, err := parseModelSettingCSV(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("parse %s: %w", candidate.path, err)
		}
		s.replaceState(records, summary, candidate.path, candidate.source)
		logger.LegacyPrintf("service.model_setting", "[ModelSetting] Loaded %d models from %s", len(records), candidate.path)
		return nil
	}

	s.replaceState(map[string]ModelSettingRecord{}, ModelSettingSummary{}, s.TargetPath(), "empty")
	return nil
}

func (s *ModelSettingService) TargetPath() string {
	dataDir := "./data"
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Pricing.DataDir) != "" {
		dataDir = strings.TrimSpace(s.cfg.Pricing.DataDir)
	}
	return filepath.Clean(filepath.Join(dataDir, modelSettingDirName, modelSettingFileName))
}

func (s *ModelSettingService) LookupModelSerialNumber(modelID string) (int, bool) {
	if s == nil {
		return 0, false
	}
	key := normalizeModelSettingID(modelID)
	if key == "" {
		return 0, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.state.recordsByID[key]
	return record.SerialNumber, ok
}

func (s *ModelSettingService) Records() []ModelSettingRecord {
	if s == nil {
		return []ModelSettingRecord{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]ModelSettingRecord, 0, len(s.state.recordsByID))
	for _, record := range s.state.recordsByID {
		records = append(records, record)
	}
	sortModelSettingRecords(records)
	return records
}

func (s *ModelSettingService) Status() ModelSettingStatus {
	if s == nil {
		return ModelSettingStatus{FileName: modelSettingFileName}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked()
}

func (s *ModelSettingService) UploadCSV(filename string, r io.Reader) (ModelSettingStatus, error) {
	if s == nil {
		return ModelSettingStatus{}, infraerrors.ServiceUnavailable("MODEL_SETTING_NOT_READY", "model setting service not ready")
	}
	if !strings.EqualFold(filepath.Ext(strings.TrimSpace(filename)), ".csv") {
		return ModelSettingStatus{}, infraerrors.BadRequest("INVALID_FILE_TYPE", "only .csv files are supported")
	}
	body, err := io.ReadAll(io.LimitReader(r, 8*1024*1024+1))
	if err != nil {
		return ModelSettingStatus{}, infraerrors.BadRequest("READ_FILE_FAILED", "failed to read uploaded file").WithCause(err)
	}
	if len(body) > 8*1024*1024 {
		return ModelSettingStatus{}, infraerrors.BadRequest("FILE_TOO_LARGE", "csv file is too large")
	}

	records, summary, err := parseModelSettingCSV(bytes.NewReader(body))
	if err != nil {
		return ModelSettingStatus{}, err
	}

	target := s.TargetPath()
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return ModelSettingStatus{}, infraerrors.InternalServer("MODEL_SETTING_WRITE_FAILED", "failed to create model setting directory").WithCause(err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".models_grouped_id_desc_*.csv")
	if err != nil {
		return ModelSettingStatus{}, infraerrors.InternalServer("MODEL_SETTING_WRITE_FAILED", "failed to create temporary csv file").WithCause(err)
	}
	tmpName := tmp.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return ModelSettingStatus{}, infraerrors.InternalServer("MODEL_SETTING_WRITE_FAILED", "failed to write temporary csv file").WithCause(err)
	}
	if err := tmp.Close(); err != nil {
		return ModelSettingStatus{}, infraerrors.InternalServer("MODEL_SETTING_WRITE_FAILED", "failed to close temporary csv file").WithCause(err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return ModelSettingStatus{}, infraerrors.InternalServer("MODEL_SETTING_WRITE_FAILED", "failed to replace model setting csv file").WithCause(err)
	}
	cleanupTmp = false

	s.replaceState(records, summary, target, "uploaded")
	return s.Status(), nil
}

func (s *ModelSettingService) replaceState(records map[string]ModelSettingRecord, summary ModelSettingSummary, sourcePath, source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = modelSettingState{
		recordsByID: records,
		summary:     summary,
		sourcePath:  sourcePath,
		source:      source,
		updatedAt:   time.Now().UTC(),
	}
}

func (s *ModelSettingService) statusLocked() ModelSettingStatus {
	updatedAt := ""
	if !s.state.updatedAt.IsZero() {
		updatedAt = s.state.updatedAt.Format(time.RFC3339)
	}
	filePath := s.state.sourcePath
	if filePath == "" {
		filePath = s.TargetPath()
	}
	return ModelSettingStatus{
		FilePath:   filePath,
		FileName:   filepath.Base(filePath),
		Source:     s.state.source,
		ModelCount: len(s.state.recordsByID),
		UpdatedAt:  updatedAt,
		Summary:    s.state.summary,
	}
}

type modelSettingLoadCandidate struct {
	path   string
	source string
}

func (s *ModelSettingService) loadCandidates() []modelSettingLoadCandidate {
	seen := map[string]struct{}{}
	add := func(out []modelSettingLoadCandidate, path, source string) []modelSettingLoadCandidate {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return out
		}
		seen[path] = struct{}{}
		return append(out, modelSettingLoadCandidate{path: path, source: source})
	}

	var out []modelSettingLoadCandidate
	out = add(out, s.TargetPath(), "data")
	out = add(out, modelSettingFileName, "fallback")
	out = add(out, filepath.Join("..", modelSettingFileName), "fallback")
	return out
}

func parseModelSettingCSV(r io.Reader) (map[string]ModelSettingRecord, ModelSettingSummary, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, ModelSettingSummary{}, infraerrors.BadRequest("INVALID_CSV", "csv file is empty")
		}
		return nil, ModelSettingSummary{}, infraerrors.BadRequest("INVALID_CSV", "failed to read csv header").WithCause(err)
	}
	summary := ModelSettingSummary{HeaderRowCount: 1}
	columns := make(map[string]int, len(header))
	for i, name := range header {
		columns[strings.ToLower(strings.TrimSpace(name))] = i
	}
	serialIdx, ok := columns["serial_number"]
	if !ok {
		return nil, summary, infraerrors.BadRequest("INVALID_CSV", "csv must contain serial_number column")
	}
	idIdx, ok := columns["id"]
	if !ok {
		return nil, summary, infraerrors.BadRequest("INVALID_CSV", "csv must contain id column")
	}

	records := make(map[string]ModelSettingRecord)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, summary, infraerrors.BadRequest("INVALID_CSV", "failed to read csv row").WithCause(err)
		}
		if isBlankCSVRow(row) {
			summary.SkippedRows++
			continue
		}
		summary.TotalRows++
		if serialIdx >= len(row) || idIdx >= len(row) {
			return nil, summary, infraerrors.BadRequest("INVALID_CSV", "csv row is missing required columns")
		}
		rawSerial := strings.TrimSpace(row[serialIdx])
		serial, err := strconv.Atoi(rawSerial)
		if err != nil {
			return nil, summary, infraerrors.BadRequest("INVALID_CSV", fmt.Sprintf("invalid serial_number %q", rawSerial)).WithCause(err)
		}
		id := strings.TrimSpace(row[idIdx])
		if id == "" {
			return nil, summary, infraerrors.BadRequest("INVALID_CSV", "model id cannot be empty")
		}
		key := normalizeModelSettingID(id)
		if _, exists := records[key]; exists {
			summary.DuplicateRows++
			continue
		}
		records[key] = ModelSettingRecord{SerialNumber: serial, ID: id}
		summary.LoadedRows++
	}
	return records, summary, nil
}

func normalizeModelSettingID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func isBlankCSVRow(row []string) bool {
	for _, item := range row {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}
	return true
}

func sortModelSettingRecords(records []ModelSettingRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].SerialNumber != records[j].SerialNumber {
			return records[i].SerialNumber > records[j].SerialNumber
		}
		return strings.ToLower(records[i].ID) < strings.ToLower(records[j].ID)
	})
}
