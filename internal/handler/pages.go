package handler

import (
	"embed"
	"errors"
	"html/template"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/emilburganov/link-shortener/internal/link"
)

//go:embed pages/*.html
var pagesFS embed.FS

var pageTemplates = template.Must(template.ParseFS(pagesFS, "pages/*.html"))

type unavailablePage struct {
	Status  int
	Heading string
	Text    string
	HomeURL string
}

var pageByError = []struct {
	err  error
	page unavailablePage
}{
	{
		err: link.ErrNotFound,
		page: unavailablePage{
			Status:  404,
			Heading: "Ссылка не найдена",
			Text:    "Проверьте правильность адреса.",
		},
	},
	{
		err: link.ErrGone,
		page: unavailablePage{
			Status:  410,
			Heading: "Ссылка больше не действует",
			Text:    "Исчерпан лимит переходов или истек срок действия.",
		},
	},
}

func wantsHTML(c *gin.Context) bool {
	return strings.Contains(c.GetHeader("Accept"), "text/html")
}

func (h *Handler) writeRedirectError(c *gin.Context, err error) {
	if wantsHTML(c) {
		for _, p := range pageByError {
			if errors.Is(err, p.err) {
				page := p.page
				page.HomeURL = h.frontendURL

				c.HTML(page.Status, "unavailable.html", page)

				return
			}
		}
	}

	writeError(c, err)
}
