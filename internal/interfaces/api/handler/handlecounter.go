package handler

import (
	"net/http"
)

func (s *Handler) CounterHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t != "Counter" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	s.ContentHandler(res, req)
}
