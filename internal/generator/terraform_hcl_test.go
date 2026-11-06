package generator

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

// These checks stand in for `terraform validate` and `terraform fmt -check` so
// template mistakes are caught without a Terraform binary. hclwrite.Format is
// the formatter terraform fmt is built on.

func firstDifference(want, got string) string {
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return fmt.Sprintf("line %d:\n  generated: %q\n  formatted: %q", i+1, w, g)
		}
	}
	return "no difference"
}

func assertValidHCL(t *testing.T, name, src string) {
	t.Helper()
	if _, diags := hclsyntax.ParseConfig([]byte(src), name, hcl.InitialPos); diags.HasErrors() {
		t.Fatalf("%s is not valid HCL: %s", name, diags.Error())
	}
	if formatted := string(hclwrite.Format([]byte(src))); formatted != src {
		t.Errorf("%s is not terraform-fmt clean, first difference at %s", name, firstDifference(src, formatted))
	}
}

var (
	variableDeclaration = regexp.MustCompile(`(?m)^variable\s+"(\w+)"`)
	variableReference   = regexp.MustCompile(`\bvar\.(\w+)`)
)

func assertVariablesDeclared(t *testing.T, files map[string]FileDef) {
	t.Helper()
	declared := map[string]bool{}
	for _, m := range variableDeclaration.FindAllStringSubmatch(files["variables.tf"].Content, -1) {
		declared[m[1]] = true
	}
	for _, m := range variableReference.FindAllStringSubmatch(files["main.tf"].Content, -1) {
		if !declared[m[1]] {
			t.Errorf("main.tf references var.%s, which variables.tf does not declare", m[1])
		}
	}
}

var terraformProviders = []string{"aws", "gcp", "azure"}

// stackSamples spans the smallest and largest selectable stacks.
var stackSamples = map[string]config.ProjectConfig{
	"empty": {},
	"go-only": {
		Frontend: config.FrontendNone, Backend: config.BackendGo, Database: config.DatabaseNone, Cache: config.CacheNone,
	},
	"typical": {
		Frontend: config.FrontendReact, Backend: config.BackendGo, Database: config.DatabasePostgres, Cache: config.CacheRedis,
	},
	"typical-monitored": {
		Frontend: config.FrontendReact, Backend: config.BackendGo, Database: config.DatabasePostgres, Cache: config.CacheRedis,
		Nginx: true, Monitoring: true,
	},
	"everything": {
		Frontend: config.FrontendVue, Backend: config.BackendJava, Database: config.DatabaseMySQL, Cache: config.CacheRedis,
		Nginx: true, Moto: true, Monitoring: true, CICD: true,
	},
}

func TestGeneratedTerraformIsValidHCL(t *testing.T) {
	for _, provider := range terraformProviders {
		for name, cfg := range stackSamples {
			t.Run(provider+"/"+name, func(t *testing.T) {
				files, err := GenerateTerraform(provider, cfg)
				if err != nil {
					t.Fatal(err)
				}
				assertValidHCL(t, provider+"/main.tf", files["main.tf"].Content)
				assertValidHCL(t, provider+"/variables.tf", files["variables.tf"].Content)
				assertVariablesDeclared(t, files)
			})
		}
	}
}
