package token

import (
	"flag"
	"github.com/mdfriday/hugoverse/pkg/hmac"
)

var (
	// HMAC
	signKey       = flag.String("sign-hmac-key", "MDFriday hakuna matata 789123", "form source authentication")
	SignatureHMAC = hmac.HMAC{Key: []byte(*signKey)}
)
