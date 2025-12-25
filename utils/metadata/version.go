/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2024 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: version metadata utilities
 *
**/

package metadata

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/maplerime/cl-query/utils/logging"
)

var (
	logger = logging.MustGetLogger("utils/metadata")
)

// Version indicates the program version information
type Version struct {
	ProgramName string `json:"program_name"`
	Branch      string `json:"branch"`
	Release     uint8  `json:"release"`
	Fixpack     uint8  `json:"fixpack"`
	Hotfix      uint8  `json:"hotfix"`
	BuildNumber string `json:"build_number"`
}

// ShortVersion returns a version string as 1.2.0
func (v *Version) ShortVersion() string {
	if v.Branch != "" {
		return fmt.Sprintf("%s-v%d.%d.%d", v.Branch, v.Release, v.Fixpack, v.Hotfix)
	}
	return fmt.Sprintf("v%d.%d.%d", v.Release, v.Fixpack, v.Hotfix)
}

// FullVersion ...
func (v *Version) FullVersion() string {
	return fmt.Sprintf("%s:\n Branch: %s\n Version: %s\n BuildNumber: %s\n Go version: %s\n OS/Arch: %s",
		v.ProgramName, v.Branch, v.ShortVersion(), v.BuildNumber, runtime.Version(),
		fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}

// DefaultVersion ...
func DefaultVersion(progName string, buildNumber string) *Version {
	return &Version{
		ProgramName: progName,
		Branch:      "",
		BuildNumber: buildNumber,
		Release:     0,
		Fixpack:     0,
		Hotfix:      0,
	}
}

// ParseVersion parses the version from the version file
// version file format:
// v1.0.0-20250310094559
// staging-v1.0.0-20250310094559
// <branch>-v<version>-<buildtime>
func ParseVersion(versionFile string) *Version {
	logger.Debugf("Parsing version from file: %s", versionFile)
	version := DefaultVersion("cl-query", "unknown")
	data, err := os.ReadFile(versionFile)
	if err != nil || len(data) == 0 {
		logger.Debugf("Failed to read version file: %s", err)
		return version
	}
	str := strings.TrimSpace(string(data))
	// parse version string
	verParts := strings.Split(str, "-")
	var branch, versionStr string
	switch len(verParts) {
	case 0:
		logger.Debugf("Version file is empty, returning default version")
		return version
	case 1:
		logger.Debugf("Version file is invalid, returning default version")
		return version
	case 2:
		logger.Debugf("No branch, using default branch")
		version.BuildNumber = verParts[1]
		versionStr = verParts[0]
	case 3:
		logger.Debugf("Branch found, using branch")
		version.BuildNumber = verParts[2]
		versionStr = verParts[1]
		branch = verParts[0]
	}
	version.Branch = branch
	version.Release, version.Fixpack, version.Hotfix = parseVersionStr(versionStr)
	logger.Debugf("Parsed version: %s", version.ShortVersion())
	return version
}

// parseVersionStr parses the version string
// version string format: v1.0.0
// return: release, fixpack, hotfix
func parseVersionStr(versionStr string) (uint8, uint8, uint8) {
	logger.Debugf("Parsing version string: %s", versionStr)
	parts := strings.Split(versionStr, ".")
	if len(parts) != 3 && len(parts) != 4 {
		return 0, 0, 0
	}
	parts[0] = strings.TrimPrefix(parts[0], "v")
	r, _ := strconv.Atoi(parts[0])
	f, _ := strconv.Atoi(parts[1])
	h, _ := strconv.Atoi(parts[2])
	logger.Debugf("Parsed version: %d.%d.%d", r, f, h)
	return uint8(r), uint8(f), uint8(h)
}
