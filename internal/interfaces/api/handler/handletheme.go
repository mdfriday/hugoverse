package handler

import (
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/hash"
)

func (s *Handler) ThemesHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t != "Theme" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	s.ApiContentsHandler(res, req)
}

func (s *Handler) ThemeHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t != "Theme" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	s.ContentHandler(res, req)
}

func (s *Handler) ThemeHashHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	n := q.Get("name")
	if n == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	q.Set("type", "Theme")
	q.Set("status", "")
	q.Set("hash", hash.Fields([]string{n}))
	req.URL.RawQuery = q.Encode()

	s.HashHandler(res, req)
}
