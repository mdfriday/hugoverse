package handler

import (
	"encoding/json"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/application"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/mdfriday/hugoverse/pkg/zip"
	"net/http"
	"path/filepath"
)

func (s *Handler) MDFPreviewHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t != "MDFPreview" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	s.ContentHandler(res, req)
}

func (s *Handler) DeployMDFridayPreviewHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	id := q.Get("id")
	t := q.Get("type")

	loggers.SetGlobalFields(s.newLogFields("deploy mdfriday preview"))

	hostName := req.FormValue("host_name")
	if hostName != "MDFriday Preview" {
		s.log.Errorf("Error: MDFriday Preview only supported for now")
		res.WriteHeader(http.StatusInternalServerError)
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

	sc, err := s.contentApp.GetContentObject(t, id)
	if err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("error GetContentObject: %v", err)).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	preview, ok := sc.(*valueobject.MDFPreview)
	if !ok {
		s.log.Error().WithFields(loggers.GetGlobalFields()).WithError(fmt.Errorf("error get MDFriday Preview: %v", err)).Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
	}

	absAssetPath, err := preview.AbsAssetPath(application.UploadDir())
	if err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("error getting absolute asset path: %v", err)).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	previewDir := ""
	switch preview.Type {
	case "share":
		previewDir = filepath.Join(application.PreviewDir(), s.db.UserDir(), preview.Name)
	case "sub":
		previewDir = filepath.Join(application.PreviewDir(), s.db.UserDir(), application.SubDomainFolder(), preview.Path)
	case "custom":
		previewDir = filepath.Join(application.PreviewDir(), s.db.UserDir(), application.CustomDomainFolder(), preview.Name)
	default:
		s.log.Error().WithFields(loggers.GetGlobalFields()).WithError(fmt.Errorf("unknown preview type: %s", preview.Type)).Logf("t: %s, id: %s", t, id)
	}

	if err := application.EnsureDirExists(previewDir); err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("error ensuring preview directory exists: %v", err)).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := zip.Unzip(absAssetPath, previewDir); err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("error unzipping asset: %v", err)).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	link := ""
	switch preview.Type {
	case "share":
		link = fmt.Sprintf("%s/%s/%s/%s", s.adminApp.CaddyURL(), application.PreviewFolder(), s.db.UserDir(), preview.Name)
	case "sub":
		link = preview.Path
	case "custom":
		link = s.adminApp.BaseURL() // TODO: we need to add path
	}

	jsonBytes, err := json.Marshal(link)
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
