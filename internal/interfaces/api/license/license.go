package license

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mdfriday/hugoverse/internal/domain/content/entity"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/token"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

// License middleware for checking license expiry
type License struct {
	contentApp *entity.Content
	log        loggers.Logger
}

// New creates a new License middleware
func New(contentApp *entity.Content, log loggers.Logger) *License {
	return &License{
		contentApp: contentApp,
		log:        log,
	}
}

// CheckExpiry middleware checks if the license has expired
// Returns 403 Forbidden with JSON error if license is expired
func (l *License) CheckExpiry(next http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// 1. Try to get email from token
		email, err := token.GetEmail(req)
		if err != nil {
			// No token found, let auth middleware handle it
			l.log.Debugf("License check: no token found, skipping expiry check")
			next.ServeHTTP(res, req)
			return
		}

		// 2. Convert email to license key
		// email format: xxxx-xxxx-xxxx@mdfriday.com
		// license key format: MDF-XXXX-XXXX-XXXX
		licenseKey := emailToLicenseKey(email)
		if licenseKey == "" {
			l.log.Debugf("License check: invalid email format %s", email)
			next.ServeHTTP(res, req)
			return
		}

		// 3. Query license from database
		license, err := l.contentApp.GetLicenseByKey(licenseKey)
		if err != nil {
			l.log.Warnf("License check: license not found for key %s", licenseKey)
			// License not found, let subsequent handler deal with it
			next.ServeHTTP(res, req)
			return
		}

		// 4. Check if license is expired
		if license.IsExpired() {
			l.log.Warnf("License expired: %s", licenseKey)
			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusForbidden)
			json.NewEncoder(res).Encode(map[string]interface{}{
				"success": false,
				"error":   "License has expired",
				"code":    "LICENSE_EXPIRED",
			})
			return
		}

		next.ServeHTTP(res, req)
	}
}

// emailToLicenseKey converts email back to license key
// Input: xxxx-xxxx-xxxx@mdfriday.com
// Output: MDF-XXXX-XXXX-XXXX
func emailToLicenseKey(email string) string {
	// Remove @mdfriday.com suffix
	email = strings.TrimSuffix(email, "@mdfriday.com")
	if email == "" || !strings.Contains(email, "-") {
		return ""
	}
	// Convert to uppercase and add MDF- prefix
	return fmt.Sprintf("MDF-%s", strings.ToUpper(email))
}

