package token

import (
	"github.com/nilslice/jwt"
	"time"
)

const userKey = "user"

func New(email string) (string, time.Time, error) {
	// create new token
	month := time.Now().Add(time.Hour * 24 * 30)
	claims := map[string]interface{}{
		"exp":   month.Unix(),
		userKey: email,
	}
	token, err := jwt.New(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	return token, month, nil
}
