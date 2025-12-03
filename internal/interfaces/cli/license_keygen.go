package cli

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

type licenseKeygenCmd struct {
	parent  *flag.FlagSet
	cmd     *flag.FlagSet
	keysDir *string
}

func NewLicenseKeygenCmd(parent *flag.FlagSet) (*licenseKeygenCmd, error) {
	nCmd := &licenseKeygenCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license keygen", flag.ExitOnError)
	nCmd.keysDir = nCmd.cmd.String("keys-dir", getDefaultKeysDir(), "Directory to store generated keys")

	err := nCmd.cmd.Parse(parent.Args()[2:]) // Skip "license" and "keygen"
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseKeygenCmd) Usage() {
	cmd.cmd.Usage()
}

func (cmd *licenseKeygenCmd) Run() error {
	service, err := license.NewService(*cmd.keysDir)
	if err != nil {
		return err
	}

	return service.GenerateKeys()
}

func getDefaultKeysDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./keys"
	}
	return filepath.Join(homeDir, ".mdfriday", "keys")
}

func getDefaultLicensesDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./licenses"
	}
	return filepath.Join(homeDir, ".mdfriday", "licenses")
}
