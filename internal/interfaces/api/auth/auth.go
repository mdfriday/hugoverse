package auth

import (
	"github.com/mdfriday/hugoverse/internal/interfaces/api/token"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"github.com/mdfriday/hugoverse/pkg/identity"
	"net/http"
)

type Auth struct {
	Session string
	UserId  string
}

func (a *Auth) CheckGetMethod(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			next.ServeHTTP(res, req)
			return
		case http.MethodPost:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		default:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})
}

func (a *Auth) CheckPostMethod(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		case http.MethodPost:
			next.ServeHTTP(res, req)
			return
		default:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})
}

func (a *Auth) CheckSignature(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := token.GetEmailFromSignature(req)
		if err == nil {
			next.ServeHTTP(res, req)
			return
		}

		res.WriteHeader(http.StatusUnauthorized)
	})
}

// Check is HTTP middleware to ensure the request has proper token credentials
func (a *Auth) Check(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if a.IsValid(req) {
			next.ServeHTTP(res, req)
			return
		}

		res.WriteHeader(http.StatusUnauthorized)
	})
}

func (a *Auth) CheckWithRedirect(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		redirect := req.URL.Scheme + req.URL.Host + "/admin/login"

		if a.IsValid(req) {
			next.ServeHTTP(res, req)
		} else {
			http.Redirect(res, req, redirect, http.StatusFound)
		}
	})
}

// IsValid checks if the user request is authenticated
func (a *Auth) IsValid(req *http.Request) bool {
	_, err := token.GetToken(req)
	if err != nil {
		return false
	}

	email, _ := token.GetEmail(req)
	a.Session = identity.GenerateSessionID()
	a.UserId = hash.MD5(email)
	return true
}
