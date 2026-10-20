package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/emilburganov/link-shortener/internal/link"
)

const maxBodyBytes = 8 << 10

type LinkService interface {
	Create(ctx context.Context, rawURL string, maxClicks *int64) (string, error)
}

type Handler struct {
	links LinkService
}

func New(links LinkService) *Handler {
	return &Handler{links: links}
}

func (h *Handler) Routes() *gin.Engine {
	router := gin.Default()
	router.POST("/links", h.createLink)

	return router
}

type createLinkRequest struct {
	URL       string `json:"url" binding:"required"`
	MaxClicks *int64 `json:"max_clicks"`
}

type createLinkResponse struct {
	Code string `json:"code"`
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

	code, err := h.links.Create(c.Request.Context(), req.URL, req.MaxClicks)

	switch {
	case errors.Is(err, link.ErrInvalidURL), errors.Is(err, link.ErrInvalidMaxClicks):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	case err != nil:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal error"})
	default:
		c.JSON(http.StatusCreated, createLinkResponse{Code: code})
	}
}
