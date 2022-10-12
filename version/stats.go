//go:build !linux
// +build !linux

package version

func (b BuildInfo) RecordComponentBuildTag(component string) {
}
