package version

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	buildVersion     = "unknown"
	buildGitRevision = "unknown"
	buildStatus      = "unknown"
	buildTag         = "unknown"
	buildHub         = "unknown"
	buildArch        = "unknown"
)

type BuildInfo struct {
	Version       string `json:"version"`
	GitRevision   string `json:"revision"`
	GolangVersion string `json:"golang_version"`
	BuildStatus   string `json:"status"`
	GitTag        string `json:"tag"`
}

type ServerInfo struct {
	Component string
	Info      BuildInfo
}

type MeshInfo []ServerInfo

type ProxyInfo struct {
	ID      string
	Version string
}

type DockerBuildInfo struct {
	Hub  string
	Tag  string
	Arch string
}

func NewBuildInfoFromOldString(oldOutput string) (BuildInfo, error) {
	res := BuildInfo{}

	lines := strings.Split(oldOutput, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.SplitN(line, ":", 2)
		if fields != nil {
			if len(fields) != 2 {
				return BuildInfo{}, fmt.Errorf("invalid BuildInfo input, field '%s' is not valid", fields[0])
			}
			value := strings.TrimSpace(fields[1])
			switch fields[0] {
			case "Version":
				res.Version = value
			case "GitRevision":
				res.GitRevision = value
			case "GolangVersion":
				res.GolangVersion = value
			case "BuildStatus":
				res.BuildStatus = value
			case "GitTag":
				res.GitTag = value
			default:
				// Skip unknown fields, as older versions may report other fields
				continue
			}
		}
	}

	return res, nil
}

var (
	// Info exports the build version information.
	Info       BuildInfo
	DockerInfo DockerBuildInfo
)

func (b BuildInfo) String() string {
	return fmt.Sprintf("%v-%v-%v",
		b.Version,
		b.GitRevision,
		b.BuildStatus)
}

func (b BuildInfo) LongForm() string {
	return fmt.Sprintf("%#v", b)
}

func init() {
	Info = BuildInfo{
		Version:       buildVersion,
		GitRevision:   buildGitRevision,
		GolangVersion: runtime.Version(),
		BuildStatus:   buildStatus,
		GitTag:        buildTag,
	}

	DockerInfo = DockerBuildInfo{
		Hub:  buildHub,
		Tag:  buildVersion,
		Arch: buildArch,
	}
}
