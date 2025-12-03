package cli

import (
	"flag"
	"fmt"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

type licenseGenerateCmd struct {
	parent    *flag.FlagSet
	cmd       *flag.FlagSet
	keysDir   *string
	plan      *string
	count     *int
	outputDir *string
}

func NewLicenseGenerateCmd(parent *flag.FlagSet) (*licenseGenerateCmd, error) {
	nCmd := &licenseGenerateCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license generate", flag.ExitOnError)
	nCmd.keysDir = nCmd.cmd.String("keys-dir", getDefaultKeysDir(), "Directory containing the keys")
	nCmd.plan = nCmd.cmd.String("plan", "lifetime", "License plan: free, yearly, or lifetime (required)")
	nCmd.count = nCmd.cmd.Int("count", 1, "Number of licenses to generate")
	nCmd.outputDir = nCmd.cmd.String("output-dir", getDefaultLicensesDir(), "Directory to save the license files")

	err := nCmd.cmd.Parse(parent.Args()[2:]) // Skip "license" and "generate"
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseGenerateCmd) Usage() {
	fmt.Println("Usage: hugov license generate [options]")
	fmt.Println("\nOptions:")
	cmd.cmd.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  hugov license generate -plan lifetime -count 5")
	fmt.Println("  hugov license generate -plan yearly -count 10")
	fmt.Println("  hugov license generate -plan free -count 3 -output-dir /custom/path")
}

func (cmd *licenseGenerateCmd) Run() error {
	// Validate count
	if *cmd.count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}

	// Validate plan
	var planType license.LicensePlan
	switch *cmd.plan {
	case "free":
		planType = license.PlanFree
	case "yearly":
		planType = license.PlanYearly
	case "lifetime":
		planType = license.PlanLifetime
	default:
		return fmt.Errorf("invalid plan: %s. Must be one of: free, yearly, lifetime", *cmd.plan)
	}

	// Create license request
	req := &license.LicenseRequest{
		Plan:  planType,
		Count: *cmd.count,
	}

	// Create service
	service, err := license.NewServiceWithPaths(*cmd.keysDir, *cmd.outputDir)
	if err != nil {
		return err
	}

	// Generate licenses (pass empty string to use service's default directory)
	return service.GenerateLicenses(req, "")
}
