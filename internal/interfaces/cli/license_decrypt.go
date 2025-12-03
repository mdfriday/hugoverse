package cli

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

type licenseDecryptCmd struct {
	parent        *flag.FlagSet
	cmd           *flag.FlagSet
	keysDir       *string
	licensesDir   *string
	encryptedFile *string
	licenseFile   *string
	outputDir     *string
}

func NewLicenseDecryptCmd(parent *flag.FlagSet) (*licenseDecryptCmd, error) {
	nCmd := &licenseDecryptCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license decrypt", flag.ExitOnError)
	nCmd.keysDir = nCmd.cmd.String("keys-dir", getDefaultKeysDir(), "Directory containing the keys")
	nCmd.licensesDir = nCmd.cmd.String("licenses-dir", getDefaultLicensesDir(), "Directory containing the licenses")
	nCmd.encryptedFile = nCmd.cmd.String("encrypted", "", "Encrypted file to decrypt (required)")
	nCmd.licenseFile = nCmd.cmd.String("license", "", "License file containing the payload (required)")
	nCmd.outputDir = nCmd.cmd.String("output-dir", "./decrypted", "Directory to save decrypted files")

	err := nCmd.cmd.Parse(parent.Args()[2:]) // Skip "license" and "decrypt"
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseDecryptCmd) Usage() {
	fmt.Println("Usage: hugov license decrypt [options]")
	fmt.Println("\nOptions:")
	cmd.cmd.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  hugov license decrypt -encrypted ./theme.json.enc -license ./activated_license.mdf.license")
	fmt.Println("  hugov license decrypt -encrypted ./template.html.enc -license ./my_license.json")
}

func (cmd *licenseDecryptCmd) Run() error {
	// Validate required parameters
	if *cmd.encryptedFile == "" {
		cmd.Usage()
		return fmt.Errorf("encrypted file is required")
	}

	if *cmd.licenseFile == "" {
		cmd.Usage()
		return fmt.Errorf("license file is required")
	}

	// Create service
	service, err := license.NewServiceWithPaths(*cmd.keysDir, *cmd.licensesDir)
	if err != nil {
		return err
	}

	// Read license file
	licenseData, err := ioutil.ReadFile(*cmd.licenseFile)
	if err != nil {
		return fmt.Errorf("failed to read license file: %w", err)
	}

	// Parse license
	var licenseObj license.License
	if err := json.Unmarshal(licenseData, &licenseObj); err != nil {
		return fmt.Errorf("failed to parse license: %w", err)
	}

	// Decode payload
	payloadBytes, err := base64.StdEncoding.DecodeString(licenseObj.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload license.LicensePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	// Read encrypted file
	encryptedData, err := ioutil.ReadFile(*cmd.encryptedFile)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Decrypt content using license
	decryptedData, err := service.DecryptContentWithLicense(&payload, encryptedData)
	if err != nil {
		return err
	}

	// Create output directory
	if err := ensureDir(*cmd.outputDir); err != nil {
		return err
	}

	// Save decrypted file
	encryptedFileName := filepath.Base(*cmd.encryptedFile)
	// Remove .enc extension and level suffix if present
	decryptedFileName := encryptedFileName
	if filepath.Ext(encryptedFileName) == ".enc" {
		decryptedFileName = encryptedFileName[:len(encryptedFileName)-4]
		// Also remove level suffix like .basic or .premium
		if strings.HasSuffix(decryptedFileName, ".basic") {
			decryptedFileName = decryptedFileName[:len(decryptedFileName)-6]
		} else if strings.HasSuffix(decryptedFileName, ".premium") {
			decryptedFileName = decryptedFileName[:len(decryptedFileName)-8]
		}
	} else {
		decryptedFileName = encryptedFileName + ".decrypted"
	}
	
	decryptedPath := filepath.Join(*cmd.outputDir, decryptedFileName)

	if err := ioutil.WriteFile(decryptedPath, decryptedData, 0644); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	fmt.Println("✅ Content decrypted successfully!")
	fmt.Printf("🔒 Encrypted file: %s\n", *cmd.encryptedFile)
	fmt.Printf("📄 License file: %s\n", *cmd.licenseFile)
	fmt.Printf("📁 Decrypted file: %s\n", decryptedPath)
	fmt.Printf("📋 License Key: %s\n", payload.LicenseKey)
	fmt.Printf("📱 Device ID: %s\n", payload.DeviceID)

	return nil
}
