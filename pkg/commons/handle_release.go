package commons

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/google/go-github/github"
)

// ReleaseAsset represents a release asset from GitHub
type ReleaseAsset struct {
	Tag      string `json:"tag"`
	AssetId  int64  `json:"asset_id"`
	Filename string `json:"filename"`
	Url      string `json:"url"`
	Path     string `json:"path"`
}

// Interface for handle libraries on GitHub
type HandleRelease interface {
	// Name basead on version, arch, os with prefix
	FormatNameRelease(prefix, goos, goarch, version string) string
	// Check if the platform is compatible with the library and return the name of the release
	PlatformCompatible() (string, error)
	// List all releases from the repository
	ListRelease(ctx context.Context) ([]ReleaseAsset, error)
	// Get the latest release compatible with the platform
	GetLatestReleaseCompatible(ctx context.Context) (*ReleaseAsset, error)
	// Check prerequisites for the library
	Prerequisites(ctx context.Context) error
	// Download the asset from the release
	DownloadAsset(ctx context.Context, release *ReleaseAsset) (string, error)
	// Extract the asset from the archive
	ExtractAsset(archive []byte, filename string, destDir string) error
}

// Anvil implementation from HandleRelease
type AnvilRelease struct {
	Namespace      string
	Repository     string
	ConfigFilename string
	Client         *github.Client
}

type AnvilConfig struct {
	AssetAnvil  ReleaseAsset `json:"asset_anvil"`
	LatestCheck string       `json:"latest_check"`
}

func RunCommandOnce(stdCtx context.Context, cmd *exec.Cmd) ([]byte, error) {
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("Don't start the command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Don't get the output: %w", err)
	}
	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("Don't wait the command: %w", err)
	}
	return output, nil
}

// Install release
func HandleReleaseExecution(stdCtx context.Context, release HandleRelease) (string, error) {
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	latest, err := release.GetLatestReleaseCompatible(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get latest release: %w", err)
	}
	err = release.Prerequisites(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to install prerequisites: %w", err)
	}
	location, err := release.DownloadAsset(ctx, latest)
	if err != nil {
		return "", fmt.Errorf("failed to download asset: %w", err)
	}
	return location, err
}

const WINDOWS = "windows"
const X86_64 = "amd64"
const LATEST_TAG = "nightly"
