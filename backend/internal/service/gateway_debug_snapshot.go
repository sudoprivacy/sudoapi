// sudoapi: Gateway debug request snapshots.

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type DebugGatewayBodyFile struct {
	logger *log.Logger
}

func NewDebugGatewayBodyFile() (debug *DebugGatewayBodyFile) {
	debug = &DebugGatewayBodyFile{}
	path := strings.TrimSpace(os.Getenv(debugGatewayBodyEnv))
	if path == "" {
		return
	}
	if parseDebugEnvBool(path) {
		path = debugGatewayBodyDefaultFilename
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, debugGatewayBodyDefaultFilename)
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error("failed to create gateway debug log directory", "dir", dir, "error", err)
			return
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Error("failed to open gateway debug log file", "path", path, "error", err)
		return
	}
	debug.logger = log.New(f, "", 0)
	slog.Info("gateway debug logging enabled", "path", path)
	return
}

func (debug *DebugGatewayBodyFile) LogGatewaySnapshot(tag string, headers http.Header, body []byte, extra map[string]string) {
	if debug == nil || debug.logger == nil {
		return
	}

	var buf strings.Builder
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	_, _ = fmt.Fprintf(&buf, "\n========== [%s] %s ==========\n", ts, tag)

	if len(extra) > 0 {
		_, _ = fmt.Fprint(&buf, "--- context ---\n")
		extraKeys := make([]string, 0, len(extra))
		for k := range extra {
			extraKeys = append(extraKeys, k)
		}
		sort.Strings(extraKeys)
		for _, k := range extraKeys {
			_, _ = fmt.Fprintf(&buf, "  %s: %s\n", k, extra[k])
		}
	}

	_, _ = fmt.Fprint(&buf, "--- headers ---\n")
	for _, k := range sortHeadersByWireOrder(headers) {
		for _, v := range headers[k] {
			_, _ = fmt.Fprintf(&buf, "  %s: %s\n", k, safeHeaderValueForLog(k, v))
		}
	}

	_, _ = fmt.Fprint(&buf, "--- body ---\n")
	if len(body) == 0 {
		_, _ = fmt.Fprint(&buf, "  (empty)\n")
	} else {
		var pretty bytes.Buffer
		if json.Indent(&pretty, body, "  ", "  ") == nil {
			_, _ = fmt.Fprintf(&buf, "  %s\n", pretty.Bytes())
		} else {
			_, _ = fmt.Fprintf(&buf, "  %s\n", body)
		}
	}
	debug.logger.Print(buf.String())
}
