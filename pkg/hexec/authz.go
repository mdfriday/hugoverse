package hexec

import (
	"bytes"
	"encoding/json"
	"github.com/mdfriday/hugoverse/pkg/herrors"
	"github.com/mdfriday/hugoverse/pkg/hexec/parser"
	"github.com/mdfriday/hugoverse/pkg/hexec/security"
	"github.com/mdfriday/hugoverse/pkg/parser/metadecoders"
	"strings"
)

type Auth struct {
	Allow security.Whitelist `json:"allow"`
	OsEnv security.Whitelist `json:"osEnv"`
}

var DefaultAuth = Auth{
	Allow: security.MustNewWhitelist(
		"^(dart-)?sass(-embedded)?$", // sass, dart-sass, dart-sass-embedded.
		"^go$",                       // for Go Modules
		"^npx$",                      // used by all Node tools (Babel, PostCSS).
		"^postcss$",
	),
	// These have been tested to work with Hugo's external programs
	// on Windows, Linux and MacOS.
	OsEnv: security.MustNewWhitelist(`(?i)^((HTTPS?|NO)_PROXY|PATH(EXT)?|APPDATA|TE?MP|TERM|GO\w+|(XDG_CONFIG_)?HOME|USERPROFILE|SSH_AUTH_SOCK|DISPLAY|LANG|SYSTEMDRIVE)$`),
}

func (c Auth) CheckAllowedExec(name string) error {
	if !c.Allow.Accept(name) {
		return &AccessDeniedError{
			Name:     name,
			Path:     "security.exec.allow",
			Policies: ToTOML(c),
		}
	}
	return nil
}

func (c Auth) OSEnvAccept(name string) bool {
	return c.OsEnv.Accept(name)
}

// ToTOML converts c to TOML with [security] as the root.
func ToTOML(c any) string {
	sec := ToSecurityMap(c)

	var b bytes.Buffer

	if err := parser.InterfaceToConfig(sec, metadecoders.TOML, &b); err != nil {
		panic(err)
	}

	return strings.TrimSpace(b.String())
}

// ToSecurityMap converts c to a map with 'security' as the root key.
func ToSecurityMap(c any) map[string]any {
	// Take it to JSON and back to get proper casing etc.
	asJson, err := json.Marshal(c)
	herrors.Must(err)
	m := make(map[string]any)
	herrors.Must(json.Unmarshal(asJson, &m))

	// Add the root
	sec := map[string]any{
		"security": m,
	}
	return sec
}
