package handler

import (
	"encoding/json"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/application"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"net/http"
)

func (s *Handler) DeployContentHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	id := q.Get("id")
	t := q.Get("type")
	status := q.Get("status")

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
	hostToken := req.FormValue("host_token")
	root := req.FormValue("domain")

	if hostToken == "" || root == "" {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("host_token: %s, domain: %s must be set", hostToken, root)).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	d, isTaken, err := s.contentApp.ApplyDomain(id, root)
	if !isTaken && err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(err).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	} else if isTaken {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			Logf("domain already taken: %s", err.Error())
		res.WriteHeader(http.StatusConflict)
		return
	}

	pt, ok := s.contentApp.GetContentCreator(t)
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	p := pt()
	_, ok = p.(content.Deployable)
	if !ok {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("not implement item.Deployable: %s", t)).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var target string

	sc, err := s.contentApp.GetContentObject(t, id)
	if err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("error GetContentObject: %v", err)).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if site, ok := sc.(*valueobject.Site); ok {
		target = site.WorkingDir
	}

	if target == "" {
		target, err = s.contentApp.BuildTarget(t, id, status)
		if err != nil {
			s.log.Error().
				WithFields(loggers.GetGlobalFields()).
				WithError(err).
				Logf("BulidTarget t: %s, id: %s", t, id)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = application.GenerateStaticSiteWithTarget(target)
		if err != nil {
			s.log.Error().
				WithFields(loggers.GetGlobalFields()).
				WithError(err).
				Logf("GenerateStaticSiteWithTarget t: %s, id: %s", t, id)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	sd, err := s.contentApp.GetDeployment(d, hostName)
	if err != nil {
		s.log.Errorf("Error getting deployment: %v", err)
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(err).
			Logf("GetDeployment d: %s, hostName: %s", d.FullDomain(), hostName)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if hostName != "Netlify" {
		s.log.Errorf("Error: Netlify only supported for now")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = application.DeployToNetlify(target, sd, d, hostToken)
	if err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(err).
			Logf("DeployToNetlify with target: %s, d: %s, hostName: %s", target, d.FullDomain(), hostName)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := s.contentApp.UpdateContentObject(sd); err != nil {
		s.log.Errorf("Error updating deployment: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.Marshal("https://" + d.FullDomain())
	if err != nil {
		s.log.Errorf("Error marshalling token: %v", err)
		return
	}

	j, err := s.res.FmtJSON(jsonBytes)
	if err != nil {
		s.log.Errorf("Error formatting JSON: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	s.res.Json(res, j)
}
