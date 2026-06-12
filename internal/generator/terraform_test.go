package generator

import (
	"strings"
	"testing"
)

func TestGenerateTerraform_AWS(t *testing.T) {
	files, err := GenerateTerraform("aws")
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
}

func TestGenerateTerraform_GCP(t *testing.T) {
	files, err := GenerateTerraform("gcp")
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
}

func TestGenerateTerraform_Azure(t *testing.T) {
	t.Run("Azure", func(t *testing.T) {
		files, err := GenerateTerraform("azure")
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

	files, err := GenerateTerraform("azure")
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
}

func TestGenerateTerraform_Invalid(t *testing.T) {
	_, err := GenerateTerraform("invalid")
	if err == nil {
		t.Error("expected error for invalid provider")
	}
}
