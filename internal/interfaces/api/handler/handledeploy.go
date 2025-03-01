package handler

import (
	"fmt"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"net/http"
)

func (s *Handler) DeployContentHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	id := q.Get("id")
	t := q.Get("type")

	loggers.SetGlobalFields(s.newLogFields("deploy"))

	if t == "" || id == "" {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("missing type or id")).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	err := req.ParseMultipartForm(form.MaxMemory)
	if err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(err).
			Logf("error parsing deploy form with t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	hostName := req.FormValue("host_name")
	if hostName == "Netlify" {
		s.deployNetlifyHandler(res, req)
		return
	} else if hostName == "Private" {
		s.deployPrivateHandler(res, req)
		return
	}
}
