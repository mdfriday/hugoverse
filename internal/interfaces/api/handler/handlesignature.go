package handler

import (
	"encoding/json"
	"flag"
	apiFrom "github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/token"
	"github.com/mdfriday/hugoverse/pkg/hmac"
	"net/http"
)

var (
	// HMAC
	signKey  = flag.String("sign-hmac-key", "MDFriday hakuna matata 789123", "form source authentication")
	signHMac = hmac.HMAC{Key: []byte(*signKey)}
)

func (s *Handler) SignatureHandler(res http.ResponseWriter, req *http.Request) {
	email, err := token.GetEmail(req)
	if err != nil {
		s.log.Errorf("Error getting email: %v", err)
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	sign, err := signHMac.Create(email)
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
		err := req.ParseMultipartForm(apiFrom.MaxMemory) // maxMemory 4MB
		if err != nil {
			s.log.Errorf("Error parsing multipart form: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		sign := req.PostFormValue("signature")
		email := req.PostFormValue("email")

		if sign == "" || email == "" {
			s.log.Errorf("Missing required fields: signature or email")
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		valid, err := signHMac.Validate(email, sign)
		if err != nil {
			s.log.Errorf("Error validating signature: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !valid {
			s.log.Errorf("Invalid signature for email: %s", email)
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		s.postContent(res, req)
	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}
