package cli

import (
	"flag"
	"fmt"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

type licenseVerifyCmd struct {
	parent      *flag.FlagSet
	cmd         *flag.FlagSet
	keysDir     *string
	licensesDir *string
	licensePath *string
}

func NewLicenseVerifyCmd(parent *flag.FlagSet) (*licenseVerifyCmd, error) {
	nCmd := &licenseVerifyCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license verify", flag.ExitOnError)
	nCmd.keysDir = nCmd.cmd.String("keys-dir", getDefaultKeysDir(), "Directory containing the keys")
	nCmd.licensesDir = nCmd.cmd.String("licenses-dir", getDefaultLicensesDir(), "Directory containing the licenses")
	nCmd.licensePath = nCmd.cmd.String("license", "", "Path to the license file to verify (required)")

	err := nCmd.cmd.Parse(parent.Args()[2:]) // Skip "license" and "verify"
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseVerifyCmd) Usage() {
	fmt.Println("Usage: hugov license verify [options]")
	fmt.Println("\nOptions:")
	cmd.cmd.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  hugov license verify -license ./user123_lifetime.mdf.license")
}

func (cmd *licenseVerifyCmd) Run() error {
	// Validate required parameters
	if *cmd.licensePath == "" {
		cmd.Usage()
		return fmt.Errorf("license path is required")
	}

	// Create service
	service, err := license.NewServiceWithPaths(*cmd.keysDir, *cmd.licensesDir)
	if err != nil {
		return err
	}

	// Verify license
	return service.VerifyLicense(*cmd.licensePath)
}
