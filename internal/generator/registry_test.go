package generator

import (
	"io/fs"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/registry"
)

func TestRegistryTemplatesExist(t *testing.T) {
	entries, _ := fs.ReadDir(FS, "templates/scaffold/backend")
	for _, e := range entries {
		t.Logf("Found in backend dir: %s", e.Name())
	}

	for id, frontend := range registry.SupportedFrontends {
		t.Run("Frontend_"+string(id), func(t *testing.T) {
			_, err := fs.Stat(FS, frontend.TemplateDir)
			if err != nil {
				t.Errorf("Template directory %q for frontend %q does not exist in embed.FS: %v", frontend.TemplateDir, id, err)
			}
		})
	}

	for id, backend := range registry.SupportedBackends {
		t.Run("Backend_"+string(id), func(t *testing.T) {
			_, err := fs.Stat(FS, backend.TemplateDir)
			if err != nil {
				t.Errorf("Template directory %q for backend %q does not exist in embed.FS: %v", backend.TemplateDir, id, err)
			}
		})
	}
}
