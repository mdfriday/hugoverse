package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

type licenseActivateCmd struct {
	parent      *flag.FlagSet
	cmd         *flag.FlagSet
	keysDir     *string
	licensesDir *string
	licenseKey  *string
	deviceID    *string
}

func NewLicenseActivateCmd(parent *flag.FlagSet) (*licenseActivateCmd, error) {
	nCmd := &licenseActivateCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license activate", flag.ExitOnError)
	nCmd.keysDir = nCmd.cmd.String("keys-dir", getDefaultKeysDir(), "Directory containing the keys")
	nCmd.licensesDir = nCmd.cmd.String("licenses-dir", getDefaultLicensesDir(), "Directory containing the licenses")
	nCmd.licenseKey = nCmd.cmd.String("key", "", "License key to activate (required)")
	nCmd.deviceID = nCmd.cmd.String("device-id", "", "Device ID for binding (required)")

	err := nCmd.cmd.Parse(parent.Args()[2:]) // Skip "license" and "activate"
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseActivateCmd) Usage() {
	fmt.Println("Usage: hugov license activate [options]")
	fmt.Println("\nOptions:")
	cmd.cmd.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  hugov license activate -key MDF-ABCD-EFGH-JKLM -device-id my-device-123")
}

func (cmd *licenseActivateCmd) Run() error {
	// Validate required parameters
	if *cmd.licenseKey == "" {
		cmd.Usage()
		return fmt.Errorf("license key is required")
	}

	if *cmd.deviceID == "" {
		cmd.Usage()
		return fmt.Errorf("device ID is required")
	}

	// Create activation request
	req := &license.ActivationRequest{
		LicenseKey: *cmd.licenseKey,
		DeviceID:   *cmd.deviceID,
	}

	// Create service
	service, err := license.NewServiceWithPaths(*cmd.keysDir, *cmd.licensesDir)
	if err != nil {
		return err
	}

	// Activate license
	response, err := service.ActivateLicense(req)
	if err != nil {
		return err
	}

	if !response.Success {
		fmt.Printf("❌ Activation failed: %s\n", response.ErrorMsg)
		return fmt.Errorf("activation failed")
	}

	fmt.Println("✅ License activated successfully!")
	fmt.Printf("🔑 License Key: %s\n", *cmd.licenseKey)
	fmt.Printf("📱 Device ID: %s\n", *cmd.deviceID)

	// Pretty print the activated license
	licenseJSON, err := json.MarshalIndent(response.License, "", "  ")
	if err == nil {
		fmt.Printf("📄 Activated License:\n%s\n", string(licenseJSON))
	}

	// Save activated license to file for testing
	plan := "lifetime" // Default, should get from response.Detail if available
	if response.Detail != nil {
		plan = string(response.Detail.Plan)
	}
	
	activatedFileName := fmt.Sprintf("%s_%s.mdf.license", *cmd.licenseKey, plan)
	if err := ioutil.WriteFile(activatedFileName, licenseJSON, 0644); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save activated license file: %v\n", err)
	} else {
		fmt.Printf("💾 Activated license saved to: %s\n", activatedFileName)
	}

	return nil
}
