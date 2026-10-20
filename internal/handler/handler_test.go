package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/emilburganov/link-shortener/internal/link"
)

const testBaseURL = "http://short.test"

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeService struct {
	code   string
	err    error
	calls  int
	gotNew link.NewLink

	visitURL   string
	visitErr   error
	visitCalls int
	gotCode    string
}

func (f *fakeService) Create(_ context.Context, in link.NewLink) (string, error) {
	f.calls++
	f.gotNew = in

	return f.code, f.err
}

func TestCreateLink(t *testing.T) {
	five := int64(5)
	expiresAt := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	created := map[string]string{
		"code":      "abc1234",
		"short_url": testBaseURL + "/abc1234",
	}

	tests := []struct {
		name       string
		body       string
		svc        *fakeService
		wantStatus int
		wantBody   map[string]string
		wantCalls  int
		wantNew    link.NewLink
	}{
		{
			name:       "created",
			body:       `{"url":"https://example.com"}`,
			svc:        &fakeService{code: "abc1234"},
			wantStatus: http.StatusCreated,
			wantBody:   created,
			wantCalls:  1,
			wantNew:    link.NewLink{URL: "https://example.com"},
		},
		{
			name:       "created with max clicks",
			body:       `{"url":"https://example.com","max_clicks":5}`,
			svc:        &fakeService{code: "abc1234"},
			wantStatus: http.StatusCreated,
			wantBody:   created,
			wantCalls:  1,
			wantNew:    link.NewLink{URL: "https://example.com", MaxClicks: &five},
		},
		{
			name:       "created with expiry",
			body:       `{"url":"https://example.com","expires_at":"2030-01-02T03:04:05Z"}`,
			svc:        &fakeService{code: "abc1234"},
			wantStatus: http.StatusCreated,
			wantBody:   created,
			wantCalls:  1,
			wantNew:    link.NewLink{URL: "https://example.com", ExpiresAt: &expiresAt},
		},
		{
			name:       "not a json",
			body:       `not json`,
			svc:        &fakeService{},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": "invalid request body"},
		},
		{
			name:       "url is missing",
			body:       `{}`,
			svc:        &fakeService{},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": "invalid request body"},
		},
		{
			name:       "url has a wrong type",
			body:       `{"url":123}`,
			svc:        &fakeService{},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": "invalid request body"},
		},
		{
			name:       "expiry has a wrong format",
			body:       `{"url":"https://example.com","expires_at":"tomorrow"}`,
			svc:        &fakeService{},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": "invalid request body"},
		},
		{
			name:       "body is too large",
			body:       `{"url":"https://example.com/` + strings.Repeat("a", maxBodyBytes) + `"}`,
			svc:        &fakeService{},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": "invalid request body"},
		},
		{
			name:       "service rejects url",
			body:       `{"url":"nope"}`,
			svc:        &fakeService{err: link.ErrInvalidURL},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": link.ErrInvalidURL.Error()},
			wantCalls:  1,
			wantNew:    link.NewLink{URL: "nope"},
		},
		{
			name:       "service rejects max clicks",
			body:       `{"url":"https://example.com","max_clicks":0}`,
			svc:        &fakeService{err: link.ErrInvalidMaxClicks},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": link.ErrInvalidMaxClicks.Error()},
			wantCalls:  1,
			wantNew:    link.NewLink{URL: "https://example.com", MaxClicks: new(int64)},
		},
		{
			name:       "service rejects expiry",
			body:       `{"url":"https://example.com","expires_at":"2030-01-02T03:04:05Z"}`,
			svc:        &fakeService{err: link.ErrInvalidExpiresAt},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": link.ErrInvalidExpiresAt.Error()},
			wantCalls:  1,
			wantNew:    link.NewLink{URL: "https://example.com", ExpiresAt: &expiresAt},
		},
		{
			name:       "wrapped error keeps its status and hides the context",
			body:       `{"url":"https://example.com"}`,
			svc:        &fakeService{err: fmt.Errorf("insert link: %w", link.ErrInvalidURL)},
			wantStatus: http.StatusBadRequest,
			wantBody:   map[string]string{"error": link.ErrInvalidURL.Error()},
			wantCalls:  1,
			wantNew:    link.NewLink{URL: "https://example.com"},
		},
		{
			name:       "unexpected error is not leaked",
			body:       `{"url":"https://example.com"}`,
			svc:        &fakeService{err: errors.New("connection to 10.0.0.5 refused")},
			wantStatus: http.StatusInternalServerError,
			wantBody:   map[string]string{"error": "internal error"},
			wantCalls:  1,
			wantNew:    link.NewLink{URL: "https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := New(tt.svc, testBaseURL).Routes()
			req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			var got map[string]string

			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("response is not a json object: %q", rec.Body.String())
			}

			if !maps.Equal(got, tt.wantBody) {
				t.Errorf("body = %v, want %v", got, tt.wantBody)
			}

			if tt.svc.calls != tt.wantCalls {
				t.Errorf("service called %d times, want %d", tt.svc.calls, tt.wantCalls)
			}

			if !equalNewLink(tt.svc.gotNew, tt.wantNew) {
				t.Errorf("service got %+v, want %+v", tt.svc.gotNew, tt.wantNew)
			}
		})
	}
}

func equalNewLink(a, b link.NewLink) bool {
	return a.URL == b.URL &&
		equalLimit(a.MaxClicks, b.MaxClicks) &&
		equalTime(a.ExpiresAt, b.ExpiresAt)
}

func equalLimit(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}

func equalTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Equal(*b)
}

func (f *fakeService) Visit(_ context.Context, code string) (string, error) {
	f.visitCalls++
	f.gotCode = code

	return f.visitURL, f.visitErr
}

func TestRedirect(t *testing.T) {
	tests := []struct {
		name         string
		svc          *fakeService
		wantStatus   int
		wantLocation string
		wantError    string
	}{
		{
			name:         "redirects to the original url",
			svc:          &fakeService{visitURL: "https://example.com/page?x=1"},
			wantStatus:   http.StatusFound,
			wantLocation: "https://example.com/page?x=1",
		},
		{
			name:       "unknown link",
			svc:        &fakeService{visitErr: link.ErrNotFound},
			wantStatus: http.StatusNotFound,
			wantError:  "link not found",
		},
		{
			name:       "exhausted or expired link",
			svc:        &fakeService{visitErr: link.ErrGone},
			wantStatus: http.StatusGone,
			wantError:  "link is no longer available",
		},
		{
			name:       "unexpected error is not leaked",
			svc:        &fakeService{visitErr: errors.New("connection to 10.0.0.5 refused")},
			wantStatus: http.StatusInternalServerError,
			wantError:  "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := New(tt.svc, testBaseURL).Routes()
			req := httptest.NewRequest(http.MethodGet, "/abc1234", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}

			if tt.wantError != "" && !strings.Contains(rec.Body.String(), tt.wantError) {
				t.Errorf("body = %q, want it to contain %q", rec.Body.String(), tt.wantError)
			}

			if tt.svc.visitCalls != 1 || tt.svc.gotCode != "abc1234" {
				t.Errorf("Visit called %d times with %q, want once with abc1234",
					tt.svc.visitCalls, tt.svc.gotCode)
			}
		})
	}
}
