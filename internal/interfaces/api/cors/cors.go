package cors

import (
	"github.com/mdfriday/hugoverse/internal/interfaces/api/cache"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"net/http"
	"net/url"
)

type Controller interface {
	CorsDisabled() bool
	Domain() string
}

type Cors struct {
	adminApp Controller
	log      loggers.Logger
	cache    *cache.Cache
}

func New(log loggers.Logger, adminApp Controller, cache *cache.Cache) *Cors {
	return &Cors{
		adminApp: adminApp,
		log:      log,
		cache:    cache,
	}
}

// Handle wraps a HandlerFunc to respond OPTIONS requests properly
func (s *Cors) Handle(next http.HandlerFunc) http.HandlerFunc {
	return s.cache.Control(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res, cors := s.responseWithCORS(res, req)
		if !cors {
			s.log.Printf("CORS disabled")
			return
		}

		if req.Method == http.MethodOptions {
			sendPreflight(res)
			return
		}

		next.ServeHTTP(res, req)
	}))
}

// sendPreflight is used to respond to a cross-origin "OPTIONS" request
func sendPreflight(res http.ResponseWriter) {
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, If-None-Match ")
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.WriteHeader(200)
	return
}

func (s *Cors) responseWithCORS(res http.ResponseWriter, req *http.Request) (http.ResponseWriter, bool) {
	if s.adminApp.CorsDisabled() {
		// check origin matches config domain
		domain := s.adminApp.Domain()
		origin := req.Header.Get("Origin")
		u, err := url.Parse(origin)
		if err != nil {
			s.log.Printf("Error parsing URL from request Origin header: %s", origin)
			return res, false
		}

		// hack to get dev environments to bypass cors since u.Host (below) will
		// be empty, based on Go's url.Parse function
		if domain == "localhost" {
			domain = ""
		}
		origin = u.Host

		// currently, this will check for exact match. will need feedback to
		// determine if subdomains should be allowed or allow multiple domains
		// in config
		if origin == domain {
			// apply limited CORS headers and return
			res.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, If-None-Match")
			res.Header().Set("Access-Control-Allow-Origin", domain)
			return res, true
		}

		// disallow request
		res.WriteHeader(http.StatusForbidden)
		return res, false
	}

	// apply full CORS headers and return
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, If-None-Match")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	return res, true
}
