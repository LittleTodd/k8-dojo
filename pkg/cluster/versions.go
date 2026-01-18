// Package cluster provides Kubernetes cluster management functionality.
package cluster

// SupportedVersion represents a Kubernetes version with its Kind node image.
type SupportedVersion struct {
	Version   string // Kubernetes version (e.g., "v1.32.0")
	NodeImage string // Kind node image tag
	IsLatest  bool   // Whether this is the latest GA version
}

// SupportedVersions returns the list of supported Kubernetes versions.
// Always provides Latest GA and N-1 versions.
func SupportedVersions() []SupportedVersion {
	return []SupportedVersion{
		{
			Version:   "v1.32.0",
			NodeImage: "kindest/node:v1.32.0",
			IsLatest:  true,
		},
		{
			Version:   "v1.31.4",
			NodeImage: "kindest/node:v1.31.4",
			IsLatest:  false,
		},
	}
}

// LatestVersion returns the latest GA Kubernetes version.
func LatestVersion() SupportedVersion {
	for _, v := range SupportedVersions() {
		if v.IsLatest {
			return v
		}
	}
	// Fallback to first version if no latest is marked
	return SupportedVersions()[0]
}

// GetVersion returns a SupportedVersion by its version string.
// Returns the latest version if not found.
func GetVersion(version string) SupportedVersion {
	for _, v := range SupportedVersions() {
		if v.Version == version {
			return v
		}
	}
	return LatestVersion()
}
