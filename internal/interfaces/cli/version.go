package cli

import (
	"flag"
	"fmt"
	"github.com/mdfriday/hugoverse/pkg/version"
	"runtime/debug"
	"sync"
)

type versionCmd struct {
	parent *flag.FlagSet
	cmd    *flag.FlagSet
}

func NewVersionCmd(parent *flag.FlagSet) (*versionCmd, error) {
	nCmd := &versionCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("version", flag.ExitOnError)
	err := nCmd.cmd.Parse(parent.Args()[1:])
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (oc *versionCmd) Usage() {
	oc.cmd.Usage()
}

func (oc *versionCmd) Run() error {
	fmt.Println(BuildVersionString())
	return nil
}

func BuildVersionString() string {
	program := "hugoverse"

	v := "v" + version.CurrentVersion.String()

	bi := getBuildInfo()
	if bi == nil {
		return v
	}
	if bi.Revision != "" {
		v += "-" + bi.Revision
	}

	osArch := bi.GoOS + "/" + bi.GoArch

	date := bi.RevisionTime
	if date == "" {
		date = "unknown"
	}

	versionString := fmt.Sprintf("%s %s %s BuildDate=%s",
		program, v, osArch, date)

	return versionString
}

var (
	bInfo     *buildInfo
	bInfoInit sync.Once
)

type buildInfo struct {
	VersionControlSystem string
	Revision             string
	RevisionTime         string
	Modified             bool

	GoOS   string
	GoArch string

	*debug.BuildInfo
}

func getBuildInfo() *buildInfo {
	bInfoInit.Do(func() {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}

		bInfo = &buildInfo{BuildInfo: bi}

		for _, s := range bInfo.Settings {
			switch s.Key {
			case "vcs":
				bInfo.VersionControlSystem = s.Value
			case "vcs.revision":
				bInfo.Revision = s.Value
			case "vcs.time":
				bInfo.RevisionTime = s.Value
			case "vcs.modified":
				bInfo.Modified = s.Value == "true"
			case "GOOS":
				bInfo.GoOS = s.Value
			case "GOARCH":
				bInfo.GoArch = s.Value
			}
		}
	})

	return bInfo
}
