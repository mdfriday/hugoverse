package version

import (
	"fmt"
)

type Version struct {
	Major int

	Minor int

	// Increment this for bug releases
	PatchLevel int

	// HugoVersionSuffix is the suffix used in the Hugo version string.
	// It will be blank for release versions.
	Suffix string
}

func (v Version) String() string {
	return version(v.Major, v.Minor, v.PatchLevel, v.Suffix)
}

func version(major, minor, patch int, suffix string) string {
	return fmt.Sprintf("%d.%d.%d%s", major, minor, patch, suffix)
}
