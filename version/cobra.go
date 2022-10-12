package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
	"sort"
	"strings"
)

type Version struct {
	ClientVersion    *BuildInfo   `json:"clientVersion,omitempty" yaml:"clientVersion,omitempty"`
	MeshVersion      *MeshInfo    `json:"meshVersion,omitempty" yaml:"meshVersion,omitempty"`
	DataPlaneVersion *[]ProxyInfo `json:"dataPlaneVersion,omitempty" yaml:"dataPlaneVersion,omitempty"`
}

type (
	GetRemoteVersionFunc func() (*MeshInfo, error)
	GetProxyVersionFunc  func() (*[]ProxyInfo, error)
)

// CobraOptions holds options to be passed to `CobraCommandWithOptions`
type CobraOptions struct {
	// GetRemoteVersion is the function to be invoked to retrieve remote versions for
	// Istio components. Optional. If not set, the 'version' subcommand will not attempt
	// to connect to a remote side, and CLI flags such as '--remote' will be hidden.
	GetRemoteVersion GetRemoteVersionFunc
	GetProxyVersions GetProxyVersionFunc
}

func CobraCommand() *cobra.Command {
	return CobraCommandWithOptions(CobraOptions{})
}

func CobraCommandWithOptions(options CobraOptions) *cobra.Command {
	var (
		short         bool
		output        string
		remote        bool
		version       Version
		remoteVersion *MeshInfo
		serverErr     error
	)

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints out build version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "" && output != "yaml" && output != "json" {
				return errors.New(`--output must be 'yaml' or 'json'`)
			}

			version.ClientVersion = &Info

			if options.GetRemoteVersion != nil && remote {
				remoteVersion, serverErr = options.GetRemoteVersion()
				if serverErr != nil {
					return serverErr
				}
				version.MeshVersion = remoteVersion
			}
			if options.GetProxyVersions != nil && remote {
				version.DataPlaneVersion, _ = options.GetProxyVersions()
			}

			switch output {
			case "":
				if short {
					if remoteVersion != nil {
						remoteVersion = coalesceVersions(remoteVersion)
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "client version: %s\n", version.ClientVersion.Version)
						for _, remote := range *remoteVersion {
							_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s version: %s\n", remote.Component, remote.Info.Version)
						}

					} else {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", version.ClientVersion.Version)
					}
					if version.DataPlaneVersion != nil {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "data plane version: %s\n", renderProxyVersions(version.DataPlaneVersion))
					}
				} else {
					if remoteVersion != nil {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "client version: %s\n", version.ClientVersion.LongForm())
						for _, remote := range *remoteVersion {
							_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s version: %s\n", remote.Component, remote.Info.LongForm())
						}
					} else {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", version.ClientVersion.LongForm())
					}
					if version.DataPlaneVersion != nil {
						for _, proxy := range *version.DataPlaneVersion {
							_, _ = fmt.Fprintf(cmd.OutOrStdout(), "data plane version: %#v\n", proxy)
						}
					}
				}
			case "yaml":
				if marshaled, err := yaml.Marshal(&version); err == nil {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(marshaled))
				}
			case "json":
				if marshaled, err := json.MarshalIndent(&version, "", "  "); err == nil {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(marshaled))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&short, "short", "s", false, "Use --short=false to generate full version information")
	cmd.Flags().StringVarP(&output, "output", "o", "", "One of 'yaml' or 'json'.")
	if options.GetRemoteVersion != nil {
		cmd.Flags().BoolVar(&remote, "remote", false, "Use --remote=false to suppress control plane check")
	}

	return cmd
}

func coalesceVersions(remoteVersion *MeshInfo) *MeshInfo {
	if identicalVersions(*remoteVersion) {
		return &MeshInfo{
			ServerInfo{
				Component: "control plane",
				Info:      (*remoteVersion)[0].Info,
			},
		}
	}

	return remoteVersion
}

func identicalVersions(remoteVersion MeshInfo) bool {
	exemplar := remoteVersion[0].Info
	for i := 1; i < len(remoteVersion); i++ {
		candidate := (remoteVersion)[i].Info
		// Note that we don't compare GitTag, GitRevision, BuildStatus,
		// or DockerHub because released Istio versions may use the same version tag
		// but differ in those fields.
		if exemplar.Version != candidate.Version {
			return false
		}
	}

	return true
}

// renderProxyVersions produces human-readable summary of an array of sidecar Istio versions
func renderProxyVersions(pinfos *[]ProxyInfo) string {
	if len(*pinfos) == 0 {
		return "none"
	}

	versions := make(map[string][]string)
	for _, pinfo := range *pinfos {
		ids := versions[pinfo.Version]
		versions[pinfo.Version] = append(ids, pinfo.ID)
	}
	sortedVersions := make([]string, 0)
	for v := range versions {
		sortedVersions = append(sortedVersions, v)
	}
	sort.Strings(sortedVersions)
	counts := []string{}
	for _, ver := range sortedVersions {
		counts = append(counts, fmt.Sprintf("%s (%d proxies)", ver, len(versions[ver])))
	}
	return strings.Join(counts, ", ")
}
