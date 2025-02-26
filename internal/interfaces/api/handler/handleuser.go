package handler

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/token"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"log"
	"net/http"
	"strings"
)

func (s *Handler) UserRegisterHandler(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		err := req.ParseForm()
		if err != nil {
			s.log.Errorf("Error parsing login form: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		// check email & password
		email := strings.ToLower(req.FormValue("email"))
		pwd := req.FormValue("password")

		found := s.adminApp.IsUserExists(email)
		s.log.Println("[UserRegisterHandler]: ", email, found)

		if found {
			s.log.Errorf("User already exists: %v", email)
			res.WriteHeader(http.StatusConflict)
			return
		}

		_, err = s.adminApp.NewUser(email, pwd)
		if err != nil {
			s.log.Errorf("Error creating new user: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		nt, _, err := token.New(email)
		if err != nil {
			s.log.Errorf("Error creating new token: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		tokenJSON, err := json.Marshal(nt)
		if err != nil {
			s.log.Errorf("Error marshalling token: %v", err)
			return
		}

		j, err := s.res.FmtJSON(tokenJSON)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusCreated)
		s.res.Json(res, j)

	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Handler) UserLoginHandler(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		err := req.ParseForm()
		if err != nil {
			s.log.Errorf("Error parsing login form: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		// check email & password
		email := strings.ToLower(req.FormValue("email"))
		pwd := req.FormValue("password")

		err = s.adminApp.ValidateUser(email, pwd)
		if err != nil {
			s.log.Errorf("Error validating user: %v", err)
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		nt, _, err := token.New(email)
		if err != nil {
			s.log.Errorf("Error creating new token: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		tokenJSON, err := json.Marshal(nt)
		if err != nil {
			s.log.Errorf("Error marshalling token: %v", err)
			return
		}

		j, err := s.res.FmtJSON(tokenJSON)
		if err != nil {
			s.log.Errorf("Error formatting JSON: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusCreated)
		s.res.Json(res, j)

	default:
		s.log.Errorf("Method not allowed: %s", req.Method)
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Handler) UserConfigHandler(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		email, err := token.GetEmail(req)
		if err != nil {
			s.log.Errorf("Error getting email: %v", err)
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		currentUser, err := s.adminApp.GetUser(email)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			errView, err := s.adminView.Error500()
			if err != nil {
				return
			}

			res.Write(errView)
			return
		}

		allUsers, err := s.adminApp.AllUsersExcept(currentUser)
		data := map[string]interface{}{
			"User":  currentUser,
			"Users": allUsers,
		}

		s.refreshAdminFlag(req)
		adminView, err := s.adminView.UserManagementView(data)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "text/html")
		res.Write(adminView)

	case http.MethodPost:
		// create new user
		err := req.ParseMultipartForm(1024 * 1024 * 4) // maxMemory 4MB
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			errView, err := s.adminView.Error500()
			if err != nil {
				return
			}

			res.Write(errView)
			return
		}

		email := strings.ToLower(req.FormValue("email"))
		password := req.PostFormValue("password")

		if email == "" || password == "" {
			res.WriteHeader(http.StatusInternalServerError)
			errView, err := s.adminView.Error500()
			if err != nil {
				return
			}

			res.Write(errView)
			return
		}

		_, err = s.adminApp.NewUser(email, password)
		if err != nil {
			s.log.Errorf("Error creating new user: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.Redirect(res, req, req.URL.String(), http.StatusFound)

	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Handler) getUserHash(req *http.Request) (string, error) {
	email, err := token.GetEmail(req)
	if err != nil {
		s.log.Errorf("Error getting email: %v", err)
		return "", err
	}

	return hash.MD5(email), nil
}
