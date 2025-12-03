package cli

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

// LicenseEncryptLevelCmd handles encrypting content with specific access level
type LicenseEncryptLevelCmd struct {
	parent    *flag.FlagSet
	fs        *flag.FlagSet
	input     *string
	level     *string
	outputDir *string
}

// NewLicenseEncryptLevelCmd creates a new license encrypt-level command
func NewLicenseEncryptLevelCmd(parent *flag.FlagSet) (*LicenseEncryptLevelCmd, error) {
	cmd := &LicenseEncryptLevelCmd{
		parent: parent,
	}

	cmd.fs = flag.NewFlagSet("license encrypt-level", flag.ExitOnError)
	cmd.input = cmd.fs.String("input", "", "Input file to encrypt")
	cmd.level = cmd.fs.String("level", "basic", "Content access level (basic|premium)")
	cmd.outputDir = cmd.fs.String("output-dir", "./encrypted_content", "Output directory for encrypted files")

	cmd.fs.Usage = func() {
		fmt.Println("Usage: hugov license encrypt-level -input <file> -level <level> [-output-dir <dir>]")
		fmt.Println("\nEncrypts content with specified access level")
		fmt.Println("\nFlags:")
		cmd.fs.PrintDefaults()
		fmt.Println("\nLevels:")
		fmt.Println("  basic   - Can be decrypted by lifetime and yearly licenses")
		fmt.Println("  premium - Can only be decrypted by yearly licenses")
		fmt.Println("\nExample:")
		fmt.Println("  hugov license encrypt-level -input ./basic_theme.json -level basic")
		fmt.Println("  hugov license encrypt-level -input ./premium_theme.json -level premium")
	}

	// Parse remaining args (skip "license" and "encrypt-level")
	if err := cmd.fs.Parse(parent.Args()[2:]); err != nil {
		return nil, err
	}

	return cmd, nil
}

// Run executes the encrypt-level command
func (cmd *LicenseEncryptLevelCmd) Run() error {
	if *cmd.input == "" {
		cmd.fs.Usage()
		return fmt.Errorf("input file is required")
	}

	// Validate level
	var contentLevel license.ContentLevel
	switch *cmd.level {
	case "basic":
		contentLevel = license.ContentLevelBasic
	case "premium":
		contentLevel = license.ContentLevelPremium
	default:
		return fmt.Errorf("invalid level: %s (must be 'basic' or 'premium')", *cmd.level)
	}

	// Read input file
	content, err := ioutil.ReadFile(*cmd.input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Create license service
	service, err := license.NewServiceWithPaths(
		getDefaultKeysDir(),
		getDefaultLicensesDir(),
	)
	if err != nil {
		return fmt.Errorf("failed to create license service: %w", err)
	}

	// Encrypt content with specified level
	encryptedContent, err := service.EncryptContentWithLevel(content, contentLevel)
	if err != nil {
		return fmt.Errorf("failed to encrypt content: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(*cmd.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate output filename
	inputFilename := filepath.Base(*cmd.input)
	outputFilename := fmt.Sprintf("%s.%s.enc", inputFilename, *cmd.level)
	outputPath := filepath.Join(*cmd.outputDir, outputFilename)

	// Write encrypted content
	if err := ioutil.WriteFile(outputPath, encryptedContent, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	fmt.Printf("✅ Content encrypted successfully!\n")
	fmt.Printf("📄 Input: %s\n", *cmd.input)
	fmt.Printf("🔒 Output: %s\n", outputPath)
	fmt.Printf("🏷️  Level: %s\n", *cmd.level)
	
	if contentLevel == license.ContentLevelBasic {
		fmt.Printf("🔑 Access: lifetime + yearly licenses can decrypt\n")
	} else {
		fmt.Printf("🔑 Access: only yearly licenses can decrypt\n")
	}

	return nil
}
