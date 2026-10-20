package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/emilburganov/link-shortener/internal/link"
)

const maxBodyBytes = 8 << 10

type LinkService interface {
	Create(ctx context.Context, in link.NewLink) (string, error)
	Visit(ctx context.Context, code string) (string, error)
}

type Handler struct {
	links       LinkService
	baseURL     string
	frontendURL string
}

func New(links LinkService, baseURL, frontendURL string) *Handler {
	return &Handler{links: links, baseURL: baseURL, frontendURL: frontendURL}
}

func (h *Handler) Routes() *gin.Engine {
	router := gin.New()
	router.SetHTMLTemplate(pageTemplates)
	router.Use(gin.Recovery(), requestLogger(slog.Default()))
	router.POST("/links", h.createLink)
	router.GET("/_assets/:name", h.asset)
	router.GET("/:code", h.redirect)

	return router
}

type createLinkRequest struct {
	URL       string     `json:"url" binding:"required"`
	MaxClicks *int64     `json:"max_clicks"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type createLinkResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) createLink(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req createLinkRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})

		return
	}

	code, err := h.links.Create(c.Request.Context(), link.NewLink{
		URL:       req.URL,
		MaxClicks: req.MaxClicks,
		ExpiresAt: req.ExpiresAt,
	})

	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusCreated, createLinkResponse{Code: code, ShortURL: h.shortURL(code)})
}

func (h *Handler) shortURL(code string) string {
	return h.baseURL + "/" + code
}

func (h *Handler) redirect(c *gin.Context) {
	target, err := h.links.Visit(c.Request.Context(), c.Param("code"))

	if err != nil {
		h.writeRedirectError(c, err)

		return
	}

	c.Redirect(http.StatusFound, target)
}

var statusByError = []struct {
	err    error
	status int
}{
	{link.ErrInvalidURL, http.StatusBadRequest},
	{link.ErrInvalidMaxClicks, http.StatusBadRequest},
	{link.ErrInvalidExpiresAt, http.StatusBadRequest},
	{link.ErrNotFound, http.StatusNotFound},
	{link.ErrGone, http.StatusGone},
}

func writeError(c *gin.Context, err error) {
	for _, m := range statusByError {
		if errors.Is(err, m.err) {
			c.JSON(m.status, errorResponse{Error: m.err.Error()})

			return
		}
	}

	_ = c.Error(err)
	c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal error"})
}
