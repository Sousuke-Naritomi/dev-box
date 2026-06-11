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
		if !strings.Contains(mainTF, check) {
			t.Errorf("expected AWS main.tf to contain %q", check)
		}
	}
	if strings.Contains(mainTF, "ami-0bba") {
		t.Error("AMI should NOT be hardcoded, should use data source")
	}

	varsTF := files["variables.tf"]
	if !strings.Contains(varsTF, "key_name") {
		t.Error("expected variables.tf to contain key_name variable")
	}
}

func TestGenerateTerraform_GCP(t *testing.T) {
	files, err := GenerateTerraform("gcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(files["main.tf"], "google_compute_firewall") {
		t.Error("expected GCP main.tf to contain firewall rule")
	}
	if !strings.Contains(files["variables.tf"], "project_id") {
		t.Error("expected GCP variables.tf to contain project_id variable")
	}
}

func TestGenerateTerraform_Azure(t *testing.T) {
	files, err := GenerateTerraform("azure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(files["main.tf"], "network_security_group") {
		t.Error("expected Azure main.tf to contain network security group")
	}
	if !strings.Contains(files["main.tf"], "azurerm_virtual_network") {
		t.Error("expected Azure main.tf to contain virtual network")
	}
}

func TestGenerateTerraform_Invalid(t *testing.T) {
	_, err := GenerateTerraform("invalid")
	if err == nil {
		t.Error("expected error for invalid provider")
	}
}
