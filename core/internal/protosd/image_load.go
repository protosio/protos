package protosd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/imageregistry"
	appruntime "github.com/protosio/protos/internal/runtime"
)

func LoadImageArchive(configFile string, version *semver.Version, opts Options, archivePath string, imageRef string) (imageregistry.LoadedImage, error) {
	cfg := config.Load(configFile, version)
	if strings.TrimSpace(opts.DataDir) != "" {
		cfg.WorkDir = opts.DataDir
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = defaultWorkDir()
	}
	workDir, err := resolvePath(cfg.WorkDir)
	if err != nil {
		return imageregistry.LoadedImage{}, err
	}
	cfg.WorkDir = workDir
	config.Get().WorkDir = workDir
	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		return imageregistry.LoadedImage{}, fmt.Errorf("failed to create Protos directory %q: %w", cfg.WorkDir, err)
	}

	platform := appruntime.Create(nil, cfg.RuntimeEndpoint)
	if platform == nil {
		return imageregistry.LoadedImage{}, fmt.Errorf("app runtime is not available on this platform")
	}
	if err := platform.Init(); err != nil {
		return imageregistry.LoadedImage{}, err
	}
	loader, ok := platform.(appruntime.ImageLoader)
	if !ok {
		return imageregistry.LoadedImage{}, fmt.Errorf("image archive loading is not supported on this platform")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	return loader.LoadImageArchive(ctx, archivePath, imageRef)
}
