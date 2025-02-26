package identity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateSessionID generates a unique session ID
func GenerateSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// If we can't generate random bytes, use timestamp
		return fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("sess-%s", hex.EncodeToString(b))
}
