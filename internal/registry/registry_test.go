package registry

import (
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

// BackendFramework and FrontendFramework share a layout, so one check covers both.
func checkFramework(t *testing.T, kind, key string, fw BackendFramework) {
	t.Helper()
	if fw.ID != key {
		t.Errorf("registry key %q and ID %q differ", key, fw.ID)
	}
	if fw.Name == "" || strings.TrimSpace(fw.Command) == "" {
		t.Error("name and command must be set")
	}
	if parts := strings.SplitN(fw.Image, ":", 2); len(parts) != 2 || parts[1] == "" || parts[1] == "latest" {
		t.Errorf("image %q must be pinned to an explicit, non-latest tag", fw.Image)
	}
	for label, port := range map[string]int{"internal": fw.InternalPort, "image": fw.ImagePort} {
		if port < 1 || port > 65535 {
			t.Errorf("%s port %d out of range", label, port)
		}
	}
	if fw.ImagePort != 8080 {
		t.Errorf("generated images and Kubernetes probes assume port 8080, got %d", fw.ImagePort)
	}
	if fw.RunAsUser <= 0 {
		t.Errorf("RunAsUser must be a non-root uid, got %d", fw.RunAsUser)
	}
	if len(fw.Volumes) == 0 || !strings.HasPrefix(fw.Volumes[0], "./"+kind+":") {
		t.Errorf("first volume must bind-mount ./%s, got %v", kind, fw.Volumes)
	}
	if want := "templates/scaffold/" + kind + "/" + fw.ID; fw.TemplateDir != want {
		t.Errorf("TemplateDir = %q, want %q", fw.TemplateDir, want)
	}
}

func TestFrameworkDefinitionsAreConsistent(t *testing.T) {
	for key, fw := range SupportedBackends {
		t.Run("backend/"+string(key), func(t *testing.T) { checkFramework(t, "backend", string(key), fw) })
	}
	for key, fw := range SupportedFrontends {
		t.Run("frontend/"+string(key), func(t *testing.T) { checkFramework(t, "frontend", string(key), BackendFramework(fw)) })
	}
}

func TestEverySelectableStackIsRegistered(t *testing.T) {
	for _, b := range []config.BackendType{config.BackendNode, config.BackendPython, config.BackendGo, config.BackendPHP, config.BackendJava} {
		if _, ok := SupportedBackends[b]; !ok {
			t.Errorf("backend %q is selectable but not registered", b)
		}
	}
	for _, f := range []config.FrontendType{config.FrontendReact, config.FrontendVue} {
		if _, ok := SupportedFrontends[f]; !ok {
			t.Errorf("frontend %q is selectable but not registered", f)
		}
	}
	if _, ok := SupportedBackends[config.BackendNone]; ok {
		t.Error("'none' must not have a registry entry")
	}
	if _, ok := SupportedFrontends[config.FrontendNone]; ok {
		t.Error("'none' must not have a registry entry")
	}
	if len(SupportedBackends) != 5 || len(SupportedFrontends) != 2 {
		t.Errorf("unexpected registry size: %d backends, %d frontends (update this test when adding a stack)", len(SupportedBackends), len(SupportedFrontends))
	}
}
