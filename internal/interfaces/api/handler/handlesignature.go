package handler

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/token"
	"net/http"
)

func (s *Handler) SignatureHandler(res http.ResponseWriter, req *http.Request) {
	email, err := token.GetEmail(req)
	if err != nil {
		s.log.Errorf("Error getting email: %v", err)
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	sign, err := token.SignatureHMAC.Create(email)
	if err != nil {
		s.log.Errorf("Error creating signature: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data": sign,
	}

	j, err := json.Marshal(resp)
	if err != nil {
		s.log.Errorf("Error marshalling response to JSON: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	_, err = res.Write(j)
	if err != nil {
		s.log.Errorf("Error writing response: %v", err)
		return
	}
}

func (s *Handler) CTAHandler(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		s.postContent(res, req)
	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}
