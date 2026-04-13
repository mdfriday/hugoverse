package handler

import (
	"encoding/base64"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/admin"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/token"
	"log"
	"net/http"
	"strings"
	"time"
)

func (s *Handler) refreshAdminFlag(req *http.Request) {
	email, err := token.GetEmail(req)
	if err != nil {
		s.log.Errorf("Error getting email: %v", err)
	}

	if email != "" {
		// 动态读取最新的 AdminEmail 配置，避免初始化时的缓存问题
		s.adminView.RefreshAdmin(email, s.adminApp.Conf.AdminEmail)
	}
}

func (s *Handler) AdminHandler(res http.ResponseWriter, req *http.Request) {
	s.refreshAdminFlag(req)
	view, err := s.adminView.Dashboard()
	if err != nil {
		s.log.Errorf("Error rendering admin view: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/html")
	res.Write(view)
}

func (s *Handler) InitHandler(res http.ResponseWriter, req *http.Request) {
	if s.db.SystemInitComplete() {
		http.Redirect(res, req, req.URL.Scheme+req.URL.Host+"/admin", http.StatusFound)
		return
	}

	switch req.Method {
	case http.MethodGet:
		view, err := admin.SetupView(s.adminApp.Name())
		if err != nil {
			s.log.Errorf("Error rendering admin view: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "text/html")
		res.Write(view)
	case http.MethodPost:
		err := req.ParseForm()
		if err != nil {
			s.log.Errorf("Error parsing form: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		// get the site name from post to encode and use as secret
		name := []byte(req.FormValue("name") + s.adminApp.NewETage())
		secret := base64.StdEncoding.EncodeToString(name)
		req.Form.Set("client_secret", secret)

		// generate an Etag to use for response caching
		etag := s.adminApp.NewETage()
		req.Form.Set("etag", etag)

		// create and save admin user
		email := strings.ToLower(req.FormValue("email"))
		password := req.FormValue("password")

		_, err = s.adminApp.NewUser(email, password)
		if err != nil {
			s.log.Errorf("Error creating new user: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		// set HTTP port which should be previously added to config cache
		req.Form.Set("http_port", s.adminApp.HttpPort())

		// set initial user email as admin_email and make config
		req.Form.Set("admin_email", email)

		err = s.adminApp.SetConfig(req.Form)
		if err != nil {
			s.log.Errorf("Error setting config: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		nt, exp, err := token.New(email)
		if err != nil {
			s.log.Errorf("Error creating JWT: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.SetCookie(res, &http.Cookie{
			Name:    "_token",
			Value:   nt,
			Expires: exp,
			Path:    "/",
		})

		redir := strings.TrimSuffix(req.URL.String(), "/init")
		http.Redirect(res, req, redir, http.StatusFound)
	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Handler) LoginHandler(res http.ResponseWriter, req *http.Request) {
	if !s.db.SystemInitComplete() {
		redir := req.URL.Scheme + req.URL.Host + "/admin/init"
		http.Redirect(res, req, redir, http.StatusFound)
		return
	}

	switch req.Method {
	case http.MethodGet:
		if s.auth.IsValid(req) {
			http.Redirect(res, req, req.URL.Scheme+req.URL.Host+"/admin", http.StatusFound)
			return
		}

		view, err := admin.Login(s.adminApp.Name())
		if err != nil {
			s.log.Errorf("Error rendering login view: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "text/html")
		res.Write(view)

	case http.MethodPost:
		if s.auth.IsValid(req) {
			http.Redirect(res, req, req.URL.Scheme+req.URL.Host+"/admin", http.StatusFound)
			return
		}

		err := req.ParseForm()
		if err != nil {
			s.log.Errorf("Error parsing login form: %v", err)
			http.Redirect(res, req, req.URL.String(), http.StatusFound)
			return
		}

		// check email & password
		email := strings.ToLower(req.FormValue("email"))
		pwd := req.FormValue("password")

		err = s.adminApp.ValidateUser(email, pwd)
		if err != nil {
			s.log.Errorf("Error validating user: %v", err)
			http.Redirect(res, req, req.URL.String(), http.StatusFound)
			return
		}

		// create new token
		nt, exp, err := token.New(email)
		if err != nil {
			s.log.Errorf("Error creating JWT: %v", err)
			http.Redirect(res, req, req.URL.String(), http.StatusFound)
			return
		}

		// add it to cookie +1 week expiration
		http.SetCookie(res, &http.Cookie{
			Name:    "_token",
			Value:   nt,
			Expires: exp,
			Path:    "/",
		})

		http.Redirect(res, req, strings.TrimSuffix(req.URL.String(), "/login"), http.StatusFound)
	}
}

func (s *Handler) LogoutHandler(res http.ResponseWriter, req *http.Request) {
	http.SetCookie(res, &http.Cookie{
		Name:    "_token",
		Expires: time.Unix(0, 0),
		Value:   "",
		Path:    "/",
	})

	http.Redirect(res, req, req.URL.Scheme+req.URL.Host+"/admin/login", http.StatusFound)
}

func (s *Handler) ConfigHandler(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		s.refreshAdminFlag(req)
		cfg, err := s.adminApp.ConfigEditor()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		adminView, err := s.adminView.SubView(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "text/html")
		res.Write(adminView)

	case http.MethodPost:
		err := req.ParseForm()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = s.adminApp.SetConfig(req.Form)
		if err != nil {
			s.log.Errorf("Error setting config: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.Redirect(res, req, req.URL.String(), http.StatusFound)

	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}
