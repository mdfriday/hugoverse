package handler

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"net/http"
)

func (s *Handler) HashHandler(res http.ResponseWriter, req *http.Request) {
	s.getContentByHash(res, req)
}

func (s *Handler) getContentByHash(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	status := q.Get("status")
	hash := q.Get("hash")

	if t == "" || hash == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	pt, ok := s.contentApp.GetContentCreator(t)
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	p := pt()

	_, ok = p.(content.Hashable)
	if !ok {
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	post, err := s.contentApp.GetContentByHash(t, hash, status)
	if err != nil {
		s.log.Errorf("Error getting content by hash %s: %v", hash, err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if post == nil {
		res.WriteHeader(http.StatusNotFound)
		s.log.Debugf("Content not found: %s %s %s", t, hash, status)
		return
	}

	err = json.Unmarshal(post, p)
	if err != nil {
		s.log.Errorf("Error unmarshalling content: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if post == nil {
		res.WriteHeader(http.StatusNotFound)
		s.log.Printf("Content not found: %s %s %s", t, hash, status)
		return
	}

	if hide(res, req, p) {
		return
	}

	push(res, req, p, post)

	j, err := s.res.FmtJSON(json.RawMessage(post))
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err = omit(res, req, p, j)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// assert hookable
	get := p
	hook, ok := get.(content.Hookable)
	if !ok {
		s.log.Errorln("[Response] error: Type", t, "does not implement item.Hookable or embed item.Item.")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// hook before response
	j, err = hook.BeforeAPIResponse(res, req, j)
	if err != nil {
		s.log.Errorln("[Response] error calling BeforeAPIResponse:", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.res.Json(res, j)

	// hook after response
	err = hook.AfterAPIResponse(res, req, j)
	if err != nil {
		s.log.Errorln("[Response] error calling AfterAPIResponse:", err)
		return
	}
}
