package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

type licenseEncryptWithLicenseCmd struct {
	parent      *flag.FlagSet
	cmd         *flag.FlagSet
	keysDir     *string
	licensesDir *string
	inputFile   *string
	licenseKey  *string
	outputDir   *string
}

func NewLicenseEncryptWithLicenseCmd(parent *flag.FlagSet) (*licenseEncryptWithLicenseCmd, error) {
	nCmd := &licenseEncryptWithLicenseCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license encrypt-with-license", flag.ExitOnError)
	nCmd.keysDir = nCmd.cmd.String("keys-dir", getDefaultKeysDir(), "Directory containing the keys")
	nCmd.licensesDir = nCmd.cmd.String("licenses-dir", getDefaultLicensesDir(), "Directory containing the licenses")
	nCmd.inputFile = nCmd.cmd.String("input", "", "Input file to encrypt (required)")
	nCmd.licenseKey = nCmd.cmd.String("license-key", "", "License key to use for encryption (required)")
	nCmd.outputDir = nCmd.cmd.String("output-dir", "./encrypted", "Directory to save encrypted files")

	err := nCmd.cmd.Parse(parent.Args()[2:]) // Skip "license" and "encrypt-with-license"
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseEncryptWithLicenseCmd) Usage() {
	fmt.Println("Usage: hugov license encrypt-with-license [options]")
	fmt.Println("\nOptions:")
	cmd.cmd.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  hugov license encrypt-with-license -input ./theme.json -license-key MDF-ABCD-EFGH-JKLM")
}

func (cmd *licenseEncryptWithLicenseCmd) Run() error {
	// Validate required parameters
	if *cmd.inputFile == "" {
		cmd.Usage()
		return fmt.Errorf("input file is required")
	}

	if *cmd.licenseKey == "" {
		cmd.Usage()
		return fmt.Errorf("license key is required")
	}

	// Create service
	service, err := license.NewServiceWithPaths(*cmd.keysDir, *cmd.licensesDir)
	if err != nil {
		return err
	}

	// Load license detail to get the CEK
	licenseDetailPath := filepath.Join(*cmd.licensesDir, fmt.Sprintf("%s.json", *cmd.licenseKey))
	detailBytes, err := ioutil.ReadFile(licenseDetailPath)
	if err != nil {
		return fmt.Errorf("failed to read license detail: %w", err)
	}

	var detail license.LicenseDetail
	if err := json.Unmarshal(detailBytes, &detail); err != nil {
		return fmt.Errorf("failed to parse license detail: %w", err)
	}

	// Read input file
	content, err := ioutil.ReadFile(*cmd.inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Encrypt content using the license's CEK
	encryptedData, err := service.EncryptContentWithLicenseKey(*cmd.licenseKey, content)
	if err != nil {
		return err
	}

	// Create output directory
	if err := ensureDir(*cmd.outputDir); err != nil {
		return err
	}

	// Save encrypted file
	inputFileName := filepath.Base(*cmd.inputFile)
	encryptedFileName := inputFileName + ".enc"
	encryptedPath := filepath.Join(*cmd.outputDir, encryptedFileName)

	if err := ioutil.WriteFile(encryptedPath, encryptedData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	fmt.Println("✅ Content encrypted successfully with license!")
	fmt.Printf("📁 Input file: %s\n", *cmd.inputFile)
	fmt.Printf("🔑 License Key: %s\n", *cmd.licenseKey)
	fmt.Printf("🔒 Encrypted file: %s\n", encryptedPath)

	return nil
}
