package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Generate Terraform files for cloud deployment",
	Long:  `Run an interactive prompt to generate main.tf for AWS, GCP, or Azure.`,
	Run: func(cmd *cobra.Command, args []string) {
		var provider string

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose your Cloud Provider").
					Options(
						huh.NewOption("AWS (Amazon Web Services)", "aws"),
						huh.NewOption("GCP (Google Cloud Platform)", "gcp"),
						huh.NewOption("Azure (Microsoft Azure)", "azure"),
					).
					Value(&provider),
			),
		)

		err := form.Run()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\n☁️  Generating Terraform configuration for %s...\n", provider)
		generateTerraform(provider)
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}

func generateTerraform(provider string) {
	var content string

	if provider == "aws" {
		content = `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "ap-northeast-1"
}

resource "aws_instance" "dev_box_server" {
  ami           = "ami-0bba69335379e17f8" # Ubuntu 22.04 LTS (Tokyo)
  instance_type = "t3.micro"

  tags = {
    Name = "dev-box-server"
  }

  user_data = <<-EOF
              #!/bin/bash
              apt-get update
              apt-get install -y docker.io docker-compose
              systemctl start docker
              systemctl enable docker
              EOF
}
`
	} else if provider == "gcp" {
		content = `terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
  }
}

provider "google" {
  project = "your-gcp-project-id"
  region  = "asia-northeast1"
  zone    = "asia-northeast1-a"
}

resource "google_compute_instance" "dev_box_server" {
  name         = "dev-box-server"
  machine_type = "e2-micro"

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
    }
  }

  network_interface {
    network = "default"
    access_config {
      // Ephemeral IP
    }
  }

  metadata_startup_script = "apt-get update && apt-get install -y docker.io docker-compose && systemctl start docker"
}
`
	} else if provider == "azure" {
		content = `terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "dev_box_rg" {
  name     = "dev-box-resources"
  location = "Japan East"
}
`
	}

	err := os.WriteFile("main.tf", []byte(content), 0644)
	if err != nil {
		fmt.Println("❌ Error writing main.tf:", err)
		return
	}
	fmt.Println("✅ Successfully generated main.tf!")
	fmt.Println("👉 Next steps: run 'terraform init' and 'terraform apply'")
}
