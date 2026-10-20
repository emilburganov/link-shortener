package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantLevel  string
		wantError  string
	}{
		{
			name:       "success is logged as info",
			path:       "/ok",
			wantStatus: http.StatusOK,
			wantLevel:  "INFO",
		},
		{
			name:       "client error is logged as warning",
			path:       "/bad",
			wantStatus: http.StatusBadRequest,
			wantLevel:  "WARN",
		},
		{
			name:       "server error is logged as error with its cause",
			path:       "/boom",
			wantStatus: http.StatusInternalServerError,
			wantLevel:  "ERROR",
			wantError:  "db down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			router := gin.New()
			router.Use(requestLogger(slog.New(slog.NewJSONHandler(&buf, nil))))
			router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
			router.GET("/bad", func(c *gin.Context) { c.Status(http.StatusBadRequest) })
			router.GET("/boom", func(c *gin.Context) {
				_ = c.Error(errors.New("db down"))
				c.Status(http.StatusInternalServerError)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			router.ServeHTTP(httptest.NewRecorder(), req)

			var entry map[string]any

			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("log line is not json: %q", buf.String())
			}

			if entry["level"] != tt.wantLevel {
				t.Errorf("level = %v, want %v", entry["level"], tt.wantLevel)
			}

			if entry["method"] != http.MethodGet {
				t.Errorf("method = %v, want GET", entry["method"])
			}

			if entry["path"] != tt.path {
				t.Errorf("path = %v, want %s", entry["path"], tt.path)
			}

			if entry["status"] != float64(tt.wantStatus) {
				t.Errorf("status = %v, want %d", entry["status"], tt.wantStatus)
			}

			got, _ := entry["error"].(string)

			if got != tt.wantError {
				t.Errorf("error = %q, want %q", got, tt.wantError)
			}
		})
	}
}
