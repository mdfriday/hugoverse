package handler

import (
	"net/http"
)

func (s *Handler) ScsHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t != "ShortCode" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	s.ApiContentsHandler(res, req)
}

func (s *Handler) ScHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t != "ShortCode" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	s.ContentHandler(res, req)
}
