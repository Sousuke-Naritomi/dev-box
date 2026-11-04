package generator

import (
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func TestGeneratedTerraformUsesCanonicalIndentation(t *testing.T) {
	for _, provider := range []string{"aws", "gcp", "azure"} {
		t.Run(provider, func(t *testing.T) {
			files, err := GenerateTerraform(provider, config.ProjectConfig{})
			if err != nil {
				t.Fatalf("GenerateTerraform failed: %v", err)
			}
			for path, file := range files {
				if strings.Contains(file.Content, "\t") {
					t.Fatalf("%s contains tab indentation and will fail terraform fmt -check", path)
				}
			}
		})
	}
}

func TestGenerateTerraform_AWS(t *testing.T) {
	files, err := GenerateTerraform("aws", config.ProjectConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := files["main.tf"]; !ok {
		t.Error("expected main.tf in output")
	}
	if _, ok := files["variables.tf"]; !ok {
		t.Error("expected variables.tf in output")
	}

	mainTF := files["main.tf"]
	checks := []string{
		"aws_security_group",
		"aws_ami",
		"key_name",
		"docker-ce",
		"docker-compose-plugin",
	}
	for _, check := range checks {
		if !strings.Contains(mainTF.Content, check) {
			t.Errorf("expected AWS main.tf to contain %q", check)
		}
	}
	if strings.Contains(mainTF.Content, "ami-0bba") {
		t.Error("AMI should NOT be hardcoded, should use data source")
	}

	if !strings.Contains(mainTF.Content, "hashicorp/aws") {
		t.Error("expected AWS provider in main.tf")
	}

	if !strings.Contains(mainTF.Content, "aws_instance") {
		t.Error("expected aws_instance resource")
	}

	varsTF := files["variables.tf"]
	if !strings.Contains(varsTF.Content, "variable \"region\"") {
		t.Error("expected region variable in variables.tf")
	}
	if !strings.Contains(varsTF.Content, "key_name") {
		t.Error("expected variables.tf to contain key_name variable")
	}
	if !strings.Contains(varsTF.Content, "allowed_ssh_cidrs") || !strings.Contains(varsTF.Content, "default     = []") {
		t.Error("expected SSH ingress to be disabled by default")
	}
	for _, securitySetting := range []string{"http_tokens   = \"required\"", "encrypted = true", "enable_public_web"} {
		if !strings.Contains(mainTF.Content+varsTF.Content, securitySetting) {
			t.Errorf("expected AWS Terraform to contain %q", securitySetting)
		}
	}
	if strings.Contains(mainTF.Content, `cidr_blocks = ["0.0.0.0/0"]`) {
		t.Error("AWS Terraform must not allow unrestricted egress by default")
	}
	for _, setting := range []string{"allowed_egress_cidrs", "enable_public_ip"} {
		if !strings.Contains(mainTF.Content+varsTF.Content, setting) {
			t.Errorf("expected AWS Terraform to contain %q", setting)
		}
	}
}

func TestGenerateTerraform_GCP(t *testing.T) {
	files, err := GenerateTerraform("gcp", config.ProjectConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(files["main.tf"].Content, "hashicorp/google") {
		t.Error("expected GCP provider")
	}
	if !strings.Contains(files["variables.tf"].Content, "project_id") {
		t.Error("expected project_id variable")
	}
	if !strings.Contains(files["main.tf"].Content, "enable-oslogin") {
		t.Error("expected GCP OS Login to be enabled")
	}
	if !strings.Contains(files["variables.tf"].Content, "allowed_ssh_cidrs") {
		t.Error("expected GCP SSH CIDR allowlist")
	}
	if !strings.Contains(files["variables.tf"].Content, "enable_public_ip") ||
		!strings.Contains(files["main.tf"].Content, "var.enable_public_ip ? [1] : []") {
		t.Error("expected GCP public IP allocation to be opt-in")
	}
}

func TestGenerateTerraform_Azure(t *testing.T) {
	t.Run("Azure", func(t *testing.T) {
		files, err := GenerateTerraform("azure", config.ProjectConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(files["main.tf"].Content, "hashicorp/azurerm") {
			t.Error("expected Azure provider")
		}
		if !strings.Contains(files["main.tf"].Content, "azurerm_resource_group") {
			t.Error("expected azurerm_resource_group")
		}
	})

	files, err := GenerateTerraform("azure", config.ProjectConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(files["main.tf"].Content, "network_security_group") {
		t.Error("expected Azure main.tf to contain network security group")
	}
	if !strings.Contains(files["main.tf"].Content, "azurerm_virtual_network") {
		t.Error("expected Azure main.tf to contain virtual network")
	}
	if !strings.Contains(files["main.tf"].Content, "secure_boot_enabled = true") {
		t.Error("expected Azure secure boot")
	}
	if !strings.Contains(files["variables.tf"].Content, "allowed_ssh_cidrs") {
		t.Error("expected Azure SSH CIDR allowlist")
	}
	if !strings.Contains(files["variables.tf"].Content, "variable \"ssh_public_key\"") ||
		!strings.Contains(files["main.tf"].Content, "public_key = var.ssh_public_key") {
		t.Error("expected Azure SSH public key to be supplied as a variable")
	}
	if strings.Contains(files["main.tf"].Content, `file("~/.ssh/id_rsa.pub")`) {
		t.Error("Azure configuration must not read a host-specific SSH key during terraform validate")
	}
	if !strings.Contains(files["terraform.tfvars.example"].Content, "ssh_public_key") {
		t.Error("expected Azure tfvars example to document ssh_public_key")
	}
}

func TestGenerateTerraform_Invalid(t *testing.T) {
	_, err := GenerateTerraform("invalid", config.ProjectConfig{})
	if err == nil {
		t.Error("expected error for invalid provider")
	}
}
