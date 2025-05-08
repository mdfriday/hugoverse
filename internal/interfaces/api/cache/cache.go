package cache

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mdfriday/hugoverse/pkg/loggers"
)

type Controller interface {
	CacheDisabled() bool
	CacheMaxAge() int64
	ETage() string
}

type Cache struct {
	adminApp Controller
	log      loggers.Logger
}

func New(log loggers.Logger, adminApp Controller) *Cache {
	return &Cache{
		adminApp: adminApp,
		log:      log,
	}
}

const (
	// DefaultMaxAge provides a default max age of 1 hour
	DefaultMaxAge = int64(60 * 60)
)

// Control sets the default cache policy on static asset responses
func (s *Cache) Control(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if s.adminApp.CacheDisabled() {
			res.Header().Add("Cache-Control", "no-cache")
			next.ServeHTTP(res, req)
		} else {
			age := s.adminApp.CacheMaxAge()
			etag := s.adminApp.ETage()
			if age == 0 {
				age = DefaultMaxAge
			}
			policy := fmt.Sprintf("max-age=%d, public", age)
			res.Header().Add("Etag", etag)
			res.Header().Add("Access-Control-Expose-Headers", "Etag")
			res.Header().Add("Cache-Control", policy)

			if match := req.Header.Get("If-None-Match"); match != "" {
				if strings.Contains(match, etag) {
					res.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, If-None-Match ")
					res.Header().Set("Access-Control-Allow-Origin", "*")
					res.WriteHeader(http.StatusNotModified)
					return
				}
			}

			next.ServeHTTP(res, req)
		}
	})
}
