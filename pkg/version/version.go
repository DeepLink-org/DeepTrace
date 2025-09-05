// Copyright (c) OpenMMLab. All rights reserved.

package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Variables injected at compile time
var (
	AgentVersion  = ""                // agent version v1.0.0
	ClientVersion = ""                // client version
	APIVersion    = "v1"              // API version v1
	FeatureFlags  = map[string]bool{} // Feature flags
	Commit        = "unknown"         // Git commit hash
	BuildTime     = "unset"           // Build time
	BuildTag      = "beta"            // Build tag dev alpha beta rc stable hotfix
)

// Version information
type VersionInfo struct {
	AgentVersion  string
	ClientVersion string
	APIVersion    string
	FeatureFlags  map[string]bool
	Commit        string
	BuildTime     string
	BuildTag      string
}

// Get version information from binary
func GetAgentVersionInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Sprintf("%s-%s (built: %s)", AgentVersion, BuildTag, BuildTime)
	}

	var revision, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}

	if revision != "" {
		if modified == "true" {
			revision += "+localmod"
		}
		return fmt.Sprintf("%s-%s (commit: %s, built: %s)",
			AgentVersion, BuildTag, revision, BuildTime)
	}

	return fmt.Sprintf("%s-%s (built: %s)", AgentVersion, BuildTag, BuildTime)
}

func GetStructuredVersion() VersionInfo {
	return VersionInfo{
		AgentVersion: AgentVersion,
		APIVersion:   APIVersion,
		FeatureFlags: FeatureFlags,
		Commit:       Commit,
		BuildTime:    BuildTime,
		BuildTag:     BuildTag,
	}
}
func GetClientVersionInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return formatFallbackVersion()
	}

	// Extract VCS information from buildinfo
	vcsInfo := extractVCSInfo(info)

	// Combine compile-time injected variables and buildinfo
	return formatFullVersion(vcsInfo)
}

// Extract VCS information (commit, modified status)
func extractVCSInfo(info *debug.BuildInfo) map[string]string {
	vcsInfo := map[string]string{
		"revision": "",
		"modified": "",
		"vcs":      "",
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			vcsInfo["revision"] = setting.Value
		case "vcs.modified":
			vcsInfo["modified"] = setting.Value
		case "vcs":
			vcsInfo["vcs"] = setting.Value
		}
	}

	return vcsInfo
}

// Format full version information
func formatFullVersion(vcsInfo map[string]string) string {
	var versionStr strings.Builder
	fmt.Println("The client version information is as follows:")
	// Basic version information
	versionStr.WriteString(fmt.Sprintf("  - Version: %s\n", ClientVersion))
	commit := Commit
	if commit == "" && vcsInfo["revision"] != "" {
		commit = vcsInfo["revision"]
		if vcsInfo["modified"] == "true" {
			commit += "+localmod"
		}
	}

	if commit != "" {
		versionStr.WriteString(fmt.Sprintf("  - Commit: %s\n", commit))
	}
	versionStr.WriteString(fmt.Sprintf("  - Build Time: %s\n", BuildTime))
	versionStr.WriteString(fmt.Sprintf("  - Build Tag: %s\n", BuildTag))

	// Other buildinfo information
	if vcsInfo["vcs"] != "" {
		versionStr.WriteString(fmt.Sprintf("VCS: %s\n", vcsInfo["vcs"]))
	}

	return versionStr.String()
}

// Format fallback version information (when buildinfo cannot be read)
func formatFallbackVersion() string {
	var versionStr strings.Builder

	versionStr.WriteString(fmt.Sprintf("Client Version: %s\n", ClientVersion))
	versionStr.WriteString(fmt.Sprintf("Build Tag: %s\n", BuildTag))
	versionStr.WriteString(fmt.Sprintf("Build Time: %s\n", BuildTime))

	if Commit != "" {
		versionStr.WriteString(fmt.Sprintf("Commit: %s\n", Commit))
	}

	return versionStr.String()
}
