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

const (
	testBaseURL   = "http://short.test"
	browserAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
)

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
			router := New(tt.svc, testBaseURL, "").Routes()
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
		accept       string
		wantStatus   int
		wantLocation string
		wantType     string
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
		{
			name:       "browser gets a page for an unknown link",
			svc:        &fakeService{visitErr: link.ErrNotFound},
			accept:     browserAccept,
			wantStatus: http.StatusNotFound,
			wantType:   "text/html",
			wantError:  "Ссылка не найдена",
		},
		{
			name:       "browser gets a page for an exhausted link",
			svc:        &fakeService{visitErr: link.ErrGone},
			accept:     browserAccept,
			wantStatus: http.StatusGone,
			wantType:   "text/html",
			wantError:  "Ссылка больше не действует",
		},
		{
			name:       "api client gets json",
			svc:        &fakeService{visitErr: link.ErrGone},
			accept:     "application/json",
			wantStatus: http.StatusGone,
			wantType:   "application/json",
			wantError:  "link is no longer available",
		},
		{
			name:       "browser still gets json for an unexpected error",
			svc:        &fakeService{visitErr: errors.New("connection to 10.0.0.5 refused")},
			accept:     browserAccept,
			wantStatus: http.StatusInternalServerError,
			wantType:   "application/json",
			wantError:  "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := New(tt.svc, testBaseURL, "").Routes()
			req := httptest.NewRequest(http.MethodGet, "/abc1234", nil)

			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}

			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, tt.wantType) {
				t.Errorf("Content-Type = %q, want prefix %q", got, tt.wantType)
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

func TestUnavailablePageHomeLink(t *testing.T) {
	tests := []struct {
		name        string
		frontendURL string
		wantLink    bool
	}{
		{
			name:        "link to the frontend when it is configured",
			frontendURL: "http://front.test",
			wantLink:    true,
		},
		{
			name:        "no link when the frontend is not configured",
			frontendURL: "",
			wantLink:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{visitErr: link.ErrNotFound}
			router := New(svc, testBaseURL, tt.frontendURL).Routes()
			req := httptest.NewRequest(http.MethodGet, "/abc1234", nil)
			req.Header.Set("Accept", browserAccept)

			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			got := strings.Contains(rec.Body.String(), `href="http://front.test"`)

			if got != tt.wantLink {
				t.Errorf("home link present = %t, want %t", got, tt.wantLink)
			}
		})
	}
}

func TestAsset(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantType   string
	}{
		{
			name:       "cyrillic font",
			path:       "/_assets/manrope-cyrillic.woff2",
			wantStatus: http.StatusOK,
			wantType:   "font/woff2",
		},
		{
			name:       "latin font",
			path:       "/_assets/manrope-latin.woff2",
			wantStatus: http.StatusOK,
			wantType:   "font/woff2",
		},
		{
			name:       "unknown file",
			path:       "/_assets/missing.woff2",
			wantStatus: http.StatusNotFound,
			wantType:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := New(&fakeService{}, testBaseURL, "").Routes()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if got := rec.Header().Get("Content-Type"); got != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantType)
			}
		})
	}
}

func TestNoRoute(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		accept    string
		wantType  string
		wantInput string
	}{
		{
			name:      "browser gets the not found page for an unknown path",
			path:      "/links/sad",
			accept:    browserAccept,
			wantType:  "text/html",
			wantInput: "Ссылка не найдена",
		},
		{
			name:      "api client gets json for an unknown path",
			path:      "/links/sad",
			accept:    "application/json",
			wantType:  "application/json",
			wantInput: `"error":"not found"`,
		},
		{
			name:      "browser gets the not found page for a deeper path",
			path:      "/a/b/c",
			accept:    browserAccept,
			wantType:  "text/html",
			wantInput: "Ссылка не найдена",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := New(&fakeService{}, testBaseURL, "").Routes()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Accept", tt.accept)

			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}

			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, tt.wantType) {
				t.Errorf("Content-Type = %q, want prefix %q", got, tt.wantType)
			}

			if !strings.Contains(rec.Body.String(), tt.wantInput) {
				t.Errorf("body = %q, want it to contain %q", rec.Body.String(), tt.wantInput)
			}
		})
	}
}
