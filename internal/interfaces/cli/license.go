package cli

import (
	"errors"
	"flag"
	"fmt"
)

type licenseCmd struct {
	parent *flag.FlagSet
	cmd    *flag.FlagSet
}

func NewLicenseCmd(parent *flag.FlagSet) (*licenseCmd, error) {
	nCmd := &licenseCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license", flag.ExitOnError)
	nCmd.cmd.Usage = func() {
		fmt.Println("Usage: hugov license [subcommand]")
		fmt.Println("\nSubcommands:")
		fmt.Println("  keygen    Generate cryptographic keys for license system")
		fmt.Println("  generate  Generate license keys")
		fmt.Println("  activate  Activate a license key with device binding")
		fmt.Println("  verify    Verify a license file")
		fmt.Println("  encrypt              Encrypt content files (for testing)")
		fmt.Println("  encrypt-with-license Encrypt content files using existing license")
		fmt.Println("  encrypt-level        Encrypt content with access level (basic|premium)")
		fmt.Println("  decrypt              Decrypt content files using license")
		fmt.Println("\nExamples:")
		fmt.Println("  hugov license keygen")
		fmt.Println("  hugov license generate -plan lifetime -count 5")
		fmt.Println("  hugov license activate -key MDF-HCWU-SE9K-3HHJ -device-id my-device-123")
		fmt.Println("  hugov license verify -license ~/.mdfriday/licenses/MDF-HCWU-SE9K-3HHJ_lifetime.mdf.license")
		fmt.Println("  hugov license encrypt -input ./theme.json")
		fmt.Println("  hugov license encrypt-with-license -input ./theme.json -license-key MDF-ABCD-EFGH-JKLM")
		fmt.Println("  hugov license encrypt-level -input ./basic_theme.json -level basic")
		fmt.Println("  hugov license encrypt-level -input ./premium_theme.json -level premium")
		fmt.Println("  hugov license decrypt -encrypted ./theme.json.enc -license ./activated.mdf.license")
	}

	err := nCmd.cmd.Parse(parent.Args()[1:])
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseCmd) Usage() {
	cmd.cmd.Usage()
}

func (cmd *licenseCmd) Run() error {
	if len(cmd.cmd.Args()) == 0 {
		cmd.Usage()
		return errors.New("please specify a license subcommand")
	}

	subCommand := cmd.cmd.Args()[0]

	switch subCommand {
	case "keygen":
		keygenCmd, err := NewLicenseKeygenCmd(cmd.parent)
		if err != nil {
			return err
		}
		return keygenCmd.Run()

	case "generate":
		generateCmd, err := NewLicenseGenerateCmd(cmd.parent)
		if err != nil {
			return err
		}
		return generateCmd.Run()

	case "activate":
		activateCmd, err := NewLicenseActivateCmd(cmd.parent)
		if err != nil {
			return err
		}
		return activateCmd.Run()

	case "verify":
		verifyCmd, err := NewLicenseVerifyCmd(cmd.parent)
		if err != nil {
			return err
		}
		return verifyCmd.Run()

	case "encrypt":
		encryptCmd, err := NewLicenseEncryptCmd(cmd.parent)
		if err != nil {
			return err
		}
		return encryptCmd.Run()

	case "encrypt-with-license":
		encryptWithLicenseCmd, err := NewLicenseEncryptWithLicenseCmd(cmd.parent)
		if err != nil {
			return err
		}
		return encryptWithLicenseCmd.Run()

	case "encrypt-level":
		encryptLevelCmd, err := NewLicenseEncryptLevelCmd(cmd.parent)
		if err != nil {
			return err
		}
		return encryptLevelCmd.Run()

	case "decrypt":
		decryptCmd, err := NewLicenseDecryptCmd(cmd.parent)
		if err != nil {
			return err
		}
		return decryptCmd.Run()

	default:
		cmd.Usage()
		return fmt.Errorf("invalid license subcommand: %s", subCommand)
	}
}
