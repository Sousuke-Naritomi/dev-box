package generator

import (
	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

// Generator defines the interface for generating project files.
// By depending on this interface, the CLI layer can easily inject
// mock generators for testing, or alternative implementations (e.g. Dry-run).
type Generator interface {
	GenerateScaffold(cfg config.ProjectConfig) (map[string]FileDef, error)
	GenerateTerraform(provider string, cfg config.ProjectConfig) (map[string]FileDef, error)
	GenerateK8sManifests(cfg config.ProjectConfig) (map[string]FileDef, error)
}

// DefaultGenerator provides the standard implementation.
type DefaultGenerator struct{}

// GenerateScaffold implements Generator.
func (g *DefaultGenerator) GenerateScaffold(cfg config.ProjectConfig) (map[string]FileDef, error) {
	return GenerateScaffold(cfg)
}

// GenerateTerraform implements Generator.
func (g *DefaultGenerator) GenerateTerraform(provider string, cfg config.ProjectConfig) (map[string]FileDef, error) {
	return GenerateTerraform(provider, cfg)
}

// GenerateK8sManifests implements Generator.
func (g *DefaultGenerator) GenerateK8sManifests(cfg config.ProjectConfig) (map[string]FileDef, error) {
	return GenerateK8sManifests(cfg)
}
