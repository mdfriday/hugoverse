package valueobject

import (
	"golang.org/x/crypto/ssh"
	"os"
)

// PasswordAuth implements password-based authentication
type PasswordAuth struct {
	Password string
}

func (p PasswordAuth) SSHAuthMethod() ssh.AuthMethod {
	return ssh.Password(p.Password)
}

// KeyAuth implements key-based authentication
type KeyAuth struct {
	PrivateKeyPath string
	Passphrase     string
}

func (k KeyAuth) SSHAuthMethod() ssh.AuthMethod {
	key, err := os.ReadFile(k.PrivateKeyPath)
	if err != nil {
		return nil
	}

	var signer ssh.Signer
	if k.Passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(k.Passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(signer)
}
