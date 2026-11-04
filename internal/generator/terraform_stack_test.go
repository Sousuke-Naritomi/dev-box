package generator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func TestTerraformSizeFollowsTheSelectedStack(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.ProjectConfig
		want map[string]string
	}{
		{
			name: "no stack information keeps the historical smallest sizes",
			cfg:  config.ProjectConfig{},
			want: map[string]string{"aws": "t3.micro", "gcp": "e2-micro", "azure": "Standard_B1s"},
		},
		{
			name: "single Go service",
			cfg:  stackSamples["go-only"],
			want: map[string]string{"aws": "t3.micro", "gcp": "e2-micro", "azure": "Standard_B1s"},
		},
		{
			name: "typical web stack needs 2 GiB",
			cfg:  stackSamples["typical"],
			want: map[string]string{"aws": "t3.small", "gcp": "e2-small", "azure": "Standard_B1ms"},
		},
		{
			name: "monitoring pushes it to 4 GiB",
			cfg:  stackSamples["typical-monitored"],
			want: map[string]string{"aws": "t3.medium", "gcp": "e2-medium", "azure": "Standard_B2s"},
		},
		{
			name: "everything enabled needs 8 GiB",
			cfg:  stackSamples["everything"],
			want: map[string]string{"aws": "t3.large", "gcp": "e2-standard-2", "azure": "Standard_B2ms"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for provider, wantType := range tt.want {
				files, err := GenerateTerraform(provider, tt.cfg)
				if err != nil {
					t.Fatal(err)
				}
				if want := `default     = "` + wantType + `"`; !strings.Contains(files["variables.tf"].Content, want) {
					t.Errorf("%s: expected %s in variables.tf", provider, want)
				}
			}
		})
	}
}

func TestEveryMemoryTierIsReachable(t *testing.T) {
	reached := map[int]bool{}
	for _, cfg := range stackSamples {
		reached[memoryTier(estimatedMemoryMiB(cfg))] = true
	}
	for tier := 0; tier < 4; tier++ {
		if !reached[tier] {
			t.Errorf("memory tier %d is never selected by any sample stack; the thresholds or sizes are dead code", tier)
		}
	}
}

func TestSizeNeverShrinksWhenServicesAreAdded(t *testing.T) {
	base := stackSamples["typical"]
	baseTier := memoryTier(estimatedMemoryMiB(base))
	for name, add := range map[string]func(*config.ProjectConfig){
		"nginx":      func(c *config.ProjectConfig) { c.Nginx = true },
		"localstack": func(c *config.ProjectConfig) { c.LocalStack = true },
		"monitoring": func(c *config.ProjectConfig) { c.Monitoring = true },
		"mysql":      func(c *config.ProjectConfig) { c.Database = config.DatabaseMySQL },
		"java":       func(c *config.ProjectConfig) { c.Backend = config.BackendJava },
	} {
		cfg := base
		add(&cfg)
		if memoryTier(estimatedMemoryMiB(cfg)) < baseTier {
			t.Errorf("adding %s must not select a smaller instance", name)
		}
	}
}

func TestStackLabelIsValidForEveryCloud(t *testing.T) {
	// GCP labels are the strictest: lowercase letters, digits, '_' and '-', at most 63 characters.
	valid := regexp.MustCompile(`^[a-z0-9_-]{1,63}$`)
	frontends := []config.FrontendType{config.FrontendReact, config.FrontendVue, config.FrontendNone}
	backends := []config.BackendType{config.BackendNode, config.BackendPython, config.BackendGo, config.BackendPHP, config.BackendJava, config.BackendNone}
	databases := []config.DatabaseType{config.DatabaseMySQL, config.DatabasePostgres, config.DatabaseNone}
	caches := []config.CacheType{config.CacheRedis, config.CacheNone}

	count := 0
	for _, f := range frontends {
		for _, b := range backends {
			for _, d := range databases {
				for _, c := range caches {
					for addons := 0; addons < 8; addons++ {
						cfg := config.ProjectConfig{
							Frontend: f, Backend: b, Database: d, Cache: c,
							Nginx: addons&1 != 0, LocalStack: addons&2 != 0, Monitoring: addons&4 != 0,
						}
						if label := stackLabel(cfg); !valid.MatchString(label) {
							t.Fatalf("stackLabel(%+v) = %q is not a valid cloud label", cfg, label)
						}
						count++
					}
				}
			}
		}
	}
	if count != 3*6*3*2*8 {
		t.Fatalf("expected to cover every combination, covered %d", count)
	}
	if got := stackLabel(config.ProjectConfig{}); got != "unspecified" {
		t.Errorf("stackLabel(empty) = %q, want unspecified", got)
	}
	if got := stackLabel(stackSamples["typical-monitored"]); got != "react-go-postgres-redis-nginx-monitoring" {
		t.Errorf("unexpected label %q", got)
	}
}

func TestTerraformCarriesTheStackLabel(t *testing.T) {
	for _, provider := range terraformProviders {
		files, err := GenerateTerraform(provider, stackSamples["typical"])
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(files["main.tf"].Content, `"react-go-postgres-redis"`) {
			t.Errorf("%s main.tf should tag the instance with the stack", provider)
		}
	}
}

func TestTerraformResourceNamesAreParameterized(t *testing.T) {
	for _, provider := range terraformProviders {
		files, err := GenerateTerraform(provider, config.ProjectConfig{})
		if err != nil {
			t.Fatal(err)
		}
		main := files["main.tf"].Content
		if strings.Contains(main, `"devbox-`) || strings.Contains(main, `["devbox"]`) {
			t.Errorf("%s main.tf still hardcodes a devbox- resource name or network tag", provider)
		}
		if !strings.Contains(main, "var.name_prefix") {
			t.Errorf("%s main.tf should name resources after var.name_prefix", provider)
		}
	}
}

func TestNamePrefixDefaultMatchesItsOwnValidation(t *testing.T) {
	for _, provider := range terraformProviders {
		files, err := GenerateTerraform(provider, config.ProjectConfig{})
		if err != nil {
			t.Fatal(err)
		}
		vars := files["variables.tf"].Content
		m := regexp.MustCompile(`regex\("([^"]+)", var\.name_prefix\)`).FindStringSubmatch(vars)
		if m == nil {
			t.Fatalf("%s: name_prefix has no validation", provider)
		}
		pattern := regexp.MustCompile(m[1])
		if !strings.Contains(vars, `default     = "devbox"`) || !pattern.MatchString("devbox") {
			t.Errorf("%s: the default name_prefix must satisfy its own validation (%s)", provider, m[1])
		}
		for _, bad := range []string{"", "a", "Devbox", "1abc", "has_underscore", "has space", strings.Repeat("a", 22)} {
			if pattern.MatchString(bad) {
				t.Errorf("%s: validation accepts %q", provider, bad)
			}
		}
	}
}

func TestTerraformGitignoreProtectsStateAndSecrets(t *testing.T) {
	files, err := GenerateTerraform("aws", config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}
	gitignore := files[".gitignore"].Content
	for _, want := range []string{".terraform/", "*.tfstate", "terraform.tfvars", "*.auto.tfvars"} {
		if !strings.Contains(gitignore, want+"\n") {
			t.Errorf(".gitignore should list %q:\n%s", want, gitignore)
		}
	}
	for _, mustCommit := range []string{".terraform.lock.hcl", "terraform.tfvars.example"} {
		for _, line := range strings.Split(gitignore, "\n") {
			if line == mustCommit {
				t.Errorf("%s must stay tracked", mustCommit)
			}
		}
	}
}

func TestAWSTfvarsExplainsEgressRequirement(t *testing.T) {
	files, err := GenerateTerraform("aws", stackSamples["typical"])
	if err != nil {
		t.Fatal(err)
	}
	example := files["terraform.tfvars.example"].Content
	for _, want := range []string{"allowed_egress_cidrs = []", "install Docker", "# instance_type = \"t3.small\"", "name_prefix"} {
		if !strings.Contains(example, want) {
			t.Errorf("tfvars example should mention %q:\n%s", want, example)
		}
	}
	if strings.Contains(files["main.tf"].Content, `cidr_blocks = ["0.0.0.0/0"]`) {
		t.Error("AWS Terraform must still not allow unrestricted egress by default")
	}
}
