package generator

import "fmt"

// GenerateTerraform generates Terraform configuration files for the specified
// cloud provider. Returns a map of filename to file content.
func GenerateTerraform(provider string) (map[string]FileDef, error) {
	files := make(map[string]FileDef)

	var tfvarsContent string

	switch provider {
	case "aws":
		tfvarsContent = "key_name = \"your-ssh-key-name\"\nallowed_ssh_cidrs = [\"203.0.113.10/32\"]\nenable_public_web = false\nallowed_web_cidrs = []"
	case "gcp":
		tfvarsContent = "project_id = \"your-gcp-project-id\"\nallowed_ssh_cidrs = [\"203.0.113.10/32\"]\nenable_public_web = false\nallowed_web_cidrs = []"
	case "azure":
		tfvarsContent = "allowed_ssh_cidrs = [\"203.0.113.10/32\"]\nenable_public_web = false\nallowed_web_cidrs = []"
	default:
		return nil, fmt.Errorf("unsupported provider: %q (supported: aws, gcp, azure)", provider)
	}

	mainContent, err := FS.ReadFile(fmt.Sprintf("templates/terraform/%s/main.tf.tmpl", provider))
	if err != nil {
		return nil, fmt.Errorf("failed to read main.tf.tmpl for %s: %w", provider, err)
	}

	variablesContent, err := FS.ReadFile(fmt.Sprintf("templates/terraform/%s/variables.tf.tmpl", provider))
	if err != nil {
		return nil, fmt.Errorf("failed to read variables.tf.tmpl for %s: %w", provider, err)
	}

	files["main.tf"] = FileDef{Content: string(mainContent), Perm: 0644}
	files["variables.tf"] = FileDef{Content: string(variablesContent), Perm: 0644}
	files["terraform.tfvars.example"] = FileDef{Content: tfvarsContent, Perm: 0644}

	return files, nil
}
