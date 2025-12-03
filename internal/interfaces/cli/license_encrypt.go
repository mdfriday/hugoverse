package cli

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

type licenseEncryptCmd struct {
	parent    *flag.FlagSet
	cmd       *flag.FlagSet
	keysDir   *string
	inputFile *string
	outputDir *string
}

func NewLicenseEncryptCmd(parent *flag.FlagSet) (*licenseEncryptCmd, error) {
	nCmd := &licenseEncryptCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license encrypt", flag.ExitOnError)
	nCmd.keysDir = nCmd.cmd.String("keys-dir", getDefaultKeysDir(), "Directory containing the keys")
	nCmd.inputFile = nCmd.cmd.String("input", "", "Input file to encrypt (required)")
	nCmd.outputDir = nCmd.cmd.String("output-dir", "./encrypted", "Directory to save encrypted files")

	err := nCmd.cmd.Parse(parent.Args()[2:]) // Skip "license" and "encrypt"
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseEncryptCmd) Usage() {
	fmt.Println("Usage: hugov license encrypt [options]")
	fmt.Println("\nOptions:")
	cmd.cmd.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  hugov license encrypt -input ./theme.json")
	fmt.Println("  hugov license encrypt -input ./template.html -output-dir ./encrypted-themes")
}

func (cmd *licenseEncryptCmd) Run() error {
	// Validate required parameters
	if *cmd.inputFile == "" {
		cmd.Usage()
		return fmt.Errorf("input file is required")
	}

	// Create service
	service, err := license.NewService(*cmd.keysDir)
	if err != nil {
		return err
	}

	// Read input file
	content, err := ioutil.ReadFile(*cmd.inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Encrypt content
	encryptedData, cek, err := service.EncryptContent(content)
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

	// Save CEK info for testing (in real scenario, this would be in license)
	cekInfoPath := filepath.Join(*cmd.outputDir, "cek_info.txt")
	cekInfo := fmt.Sprintf("CEK (for testing): %x\nOriginal file: %s\nEncrypted file: %s\n", 
		cek, *cmd.inputFile, encryptedPath)
	
	if err := ioutil.WriteFile(cekInfoPath, []byte(cekInfo), 0644); err != nil {
		return fmt.Errorf("failed to write CEK info: %w", err)
	}

	fmt.Println("✅ Content encrypted successfully!")
	fmt.Printf("📁 Input file: %s\n", *cmd.inputFile)
	fmt.Printf("🔒 Encrypted file: %s\n", encryptedPath)
	fmt.Printf("📋 CEK info: %s\n", cekInfoPath)
	fmt.Printf("🔑 CEK (hex): %x\n", cek)

	return nil
}

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}
