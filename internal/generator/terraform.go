package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

const terraformGitignore = `# Terraform state and local variable files can contain secrets.
.terraform/
*.tfstate
*.tfstate.*
crash.log
terraform.tfvars
*.auto.tfvars
`

type terraformData struct {
	InstanceType string
	Stack        string
}

// instanceTypes lists one size per memory tier (1, 2, 4 and 8 GiB) for each provider.
var instanceTypes = map[string][4]string{
	"aws":   {"t3.micro", "t3.small", "t3.medium", "t3.large"},
	"gcp":   {"e2-micro", "e2-small", "e2-medium", "e2-standard-2"},
	"azure": {"Standard_B1s", "Standard_B1ms", "Standard_B2s", "Standard_B2ms"},
}

// estimatedMemoryMiB is a rough working-set estimate for the Compose stack on a
// single VM. It only has to pick a sensible default size, which users can override.
func estimatedMemoryMiB(cfg config.ProjectConfig) int {
	total := 400 // operating system and Docker engine
	if cfg.Frontend != config.FrontendNone && cfg.Frontend != "" {
		total += 300
	}
	switch cfg.Backend {
	case config.BackendJava:
		total += 400
	case config.BackendNode, config.BackendPython, config.BackendPHP:
		total += 150
	case config.BackendGo:
		total += 100
	}
	switch cfg.Database {
	case config.DatabaseMySQL:
		total += 600
	case config.DatabasePostgres:
		total += 300
	}
	if cfg.Cache == config.CacheRedis {
		total += 50
	}
	if cfg.Nginx {
		total += 20
	}
	if cfg.LocalStack {
		total += 800
	}
	if cfg.Monitoring {
		total += 860
	}
	return total
}

func memoryTier(mib int) int {
	switch {
	case mib <= 700:
		return 0
	case mib <= 1500:
		return 1
	case mib <= 2600:
		return 2
	default:
		return 3
	}
}

// stackLabel names the selected services for use as a cloud resource tag. It
// only contains characters that are valid in GCP labels.
func stackLabel(cfg config.ProjectConfig) string {
	var parts []string
	for _, value := range []string{string(cfg.Frontend), string(cfg.Backend), string(cfg.Database), string(cfg.Cache)} {
		if value != "" && value != "none" {
			parts = append(parts, value)
		}
	}
	if cfg.Nginx {
		parts = append(parts, "nginx")
	}
	if cfg.LocalStack {
		parts = append(parts, "localstack")
	}
	if cfg.Monitoring {
		parts = append(parts, "monitoring")
	}
	if len(parts) == 0 {
		return "unspecified"
	}
	return strings.Join(parts, "-")
}

func renderTerraformTemplate(provider, name string, data terraformData) (string, error) {
	raw, err := FS.ReadFile(fmt.Sprintf("templates/terraform/%s/%s.tmpl", provider, name))
	if err != nil {
		return "", fmt.Errorf("failed to read %s.tmpl for %s: %w", name, provider, err)
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parsing %s for %s: %w", name, provider, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering %s for %s: %w", name, provider, err)
	}
	return buf.String(), nil
}

func terraformVarsExample(provider string, data terraformData) string {
	var b strings.Builder
	b.WriteString("# Copy this file to terraform.tfvars and adjust it. Do not commit terraform.tfvars.\n")
	switch provider {
	case "aws":
		b.WriteString("key_name = \"your-ssh-key-name\"\n")
		b.WriteString("enable_public_ip = false\n")
		b.WriteString("allowed_ssh_cidrs = [\"203.0.113.10/32\"]\n")
		b.WriteString("enable_public_web = false\n")
		b.WriteString("allowed_web_cidrs = []\n")
		b.WriteString("# Outbound traffic is closed by default, but the instance needs outbound HTTPS to\n")
		b.WriteString("# install Docker on first boot. Allow your NAT or proxy range here; otherwise the\n")
		b.WriteString("# installation fails (see /var/log/cloud-init-output.log on the instance).\n")
		b.WriteString("allowed_egress_cidrs = []\n")
		fmt.Fprintf(&b, "# Sized for the selected stack (%s); uncomment to override.\n# instance_type = %q\n", data.Stack, data.InstanceType)
	case "gcp":
		b.WriteString("project_id = \"your-gcp-project-id\"\n")
		b.WriteString("enable_public_ip = false\n")
		b.WriteString("allowed_ssh_cidrs = [\"203.0.113.10/32\"]\n")
		b.WriteString("enable_public_web = false\n")
		b.WriteString("allowed_web_cidrs = []\n")
		fmt.Fprintf(&b, "# Sized for the selected stack (%s); uncomment to override.\n# machine_type = %q\n", data.Stack, data.InstanceType)
	case "azure":
		b.WriteString("ssh_public_key = \"ssh-ed25519 replace-with-your-public-key\"\n")
		b.WriteString("allowed_ssh_cidrs = [\"203.0.113.10/32\"]\n")
		b.WriteString("enable_public_web = false\n")
		b.WriteString("allowed_web_cidrs = []\n")
		fmt.Fprintf(&b, "# Sized for the selected stack (%s); uncomment to override.\n# vm_size = %q\n", data.Stack, data.InstanceType)
	}
	b.WriteString("# Change name_prefix to run several environments side by side.\n# name_prefix = \"devbox\"\n")
	return b.String()
}

// GenerateTerraform generates Terraform configuration files for the specified
// cloud provider, sized and labelled for the selected stack. An empty
// ProjectConfig yields the smallest default size. Returns a map of filename to
// file content.
func GenerateTerraform(provider string, cfg config.ProjectConfig) (map[string]FileDef, error) {
	types, ok := instanceTypes[provider]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %q (supported: aws, gcp, azure)", provider)
	}
	data := terraformData{
		InstanceType: types[memoryTier(estimatedMemoryMiB(cfg))],
		Stack:        stackLabel(cfg),
	}

	mainContent, err := renderTerraformTemplate(provider, "main.tf", data)
	if err != nil {
		return nil, err
	}
	variablesContent, err := renderTerraformTemplate(provider, "variables.tf", data)
	if err != nil {
		return nil, err
	}

	return map[string]FileDef{
		"main.tf":                  {Content: mainContent, Perm: 0644},
		"variables.tf":             {Content: variablesContent, Perm: 0644},
		"terraform.tfvars.example": {Content: terraformVarsExample(provider, data), Perm: 0644},
		".gitignore":               {Content: terraformGitignore, Perm: 0644},
	}, nil
}
