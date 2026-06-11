package generator

import "fmt"

// GenerateTerraform generates Terraform configuration files for the specified
// cloud provider. Returns a map of filename to file content.
func GenerateTerraform(provider string) (map[string]FileDef, error) {
	files := make(map[string]FileDef)

	switch provider {
	case "aws":
		files["main.tf"] = FileDef{Content: awsMainTF, Perm: 0644}
		files["variables.tf"] = FileDef{Content: awsVariablesTF, Perm: 0644}
		files["terraform.tfvars.example"] = FileDef{Content: `key_name = "your-ssh-key-name"`, Perm: 0644}
	case "gcp":
		files["main.tf"] = FileDef{Content: gcpMainTF, Perm: 0644}
		files["variables.tf"] = FileDef{Content: gcpVariablesTF, Perm: 0644}
		files["terraform.tfvars.example"] = FileDef{Content: `project_id = "your-gcp-project-id"`, Perm: 0644}
	case "azure":
		files["main.tf"] = FileDef{Content: azureMainTF, Perm: 0644}
		files["variables.tf"] = FileDef{Content: azureVariablesTF, Perm: 0644}
		files["terraform.tfvars.example"] = FileDef{Content: `# No required variables for Azure default setup`, Perm: 0644}
	default:
		return nil, fmt.Errorf("unsupported provider: %q (supported: aws, gcp, azure)", provider)
	}

	return files, nil
}

const awsVariablesTF = `variable "region" {
  description = "AWS region to deploy resources"
  type        = string
  default     = "ap-northeast-1"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "key_name" {
  description = "Name of the SSH key pair to use for EC2 access"
  type        = string
}
`

const awsMainTF = `terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

data "aws_ami" "ubuntu" {
  most_recent = true
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
  owners = ["099720109477"]
}

resource "aws_security_group" "devbox" {
  name        = "devbox-sg"
  description = "Security group for dev-box instance"

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "devbox" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.devbox.id]

  user_data = <<-EOF
    #!/bin/bash
    set -euo pipefail
    apt-get update
    apt-get install -y ca-certificates curl gnupg
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable docker
    systemctl start docker
    usermod -aG docker ubuntu
  EOF

  tags = {
    Name = "devbox-instance"
  }
}

output "instance_public_ip" {
  value = aws_instance.devbox.public_ip
}
`

const gcpVariablesTF = `variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "asia-northeast1"
}

variable "zone" {
  description = "GCP zone"
  type        = string
  default     = "asia-northeast1-a"
}
`

const gcpMainTF = `terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_compute_firewall" "devbox" {
  name    = "devbox-firewall"
  network = "default"
  allow {
    protocol = "tcp"
    ports    = ["22", "80", "443"]
  }
  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["devbox"]
}

resource "google_compute_instance" "devbox" {
  name         = "devbox-instance"
  machine_type = "e2-micro"
  zone         = var.zone
  tags         = ["devbox"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
    }
  }

  network_interface {
    network = "default"
    access_config {}
  }

  metadata_startup_script = <<-EOF
    #!/bin/bash
    set -euo pipefail
    apt-get update
    apt-get install -y ca-certificates curl gnupg
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable docker
    systemctl start docker
  EOF
}

output "instance_ip" {
  value = google_compute_instance.devbox.network_interface[0].access_config[0].nat_ip
}
`

const azureVariablesTF = `variable "location" {
  description = "Azure region"
  type        = string
  default     = "japaneast"
}

variable "admin_username" {
  description = "Admin username for the VM"
  type        = string
  default     = "azureuser"
}
`

const azureMainTF = `terraform {
  required_version = ">= 1.0"
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

resource "azurerm_resource_group" "devbox" {
  name     = "devbox-rg"
  location = var.location
}

resource "azurerm_virtual_network" "devbox" {
  name                = "devbox-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.devbox.location
  resource_group_name = azurerm_resource_group.devbox.name
}

resource "azurerm_subnet" "devbox" {
  name                 = "devbox-subnet"
  resource_group_name  = azurerm_resource_group.devbox.name
  virtual_network_name = azurerm_virtual_network.devbox.name
  address_prefixes     = ["10.0.1.0/24"]
}

resource "azurerm_network_security_group" "devbox" {
  name                = "devbox-nsg"
  location            = azurerm_resource_group.devbox.location
  resource_group_name = azurerm_resource_group.devbox.name

  security_rule {
    name                       = "SSH"
    priority                   = 1001
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "HTTP"
    priority                   = 1002
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_public_ip" "devbox" {
  name                = "devbox-pip"
  location            = azurerm_resource_group.devbox.location
  resource_group_name = azurerm_resource_group.devbox.name
  allocation_method   = "Dynamic"
}

resource "azurerm_network_interface" "devbox" {
  name                = "devbox-nic"
  location            = azurerm_resource_group.devbox.location
  resource_group_name = azurerm_resource_group.devbox.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.devbox.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.devbox.id
  }
}

resource "azurerm_network_interface_security_group_association" "devbox" {
  network_interface_id      = azurerm_network_interface.devbox.id
  network_security_group_id = azurerm_network_security_group.devbox.id
}

resource "azurerm_linux_virtual_machine" "devbox" {
  name                = "devbox-vm"
  resource_group_name = azurerm_resource_group.devbox.name
  location            = azurerm_resource_group.devbox.location
  size                = "Standard_B1s"
  admin_username      = var.admin_username

  network_interface_ids = [
    azurerm_network_interface.devbox.id,
  ]

  admin_ssh_key {
    username   = var.admin_username
    public_key = file("~/.ssh/id_rsa.pub")
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts"
    version   = "latest"
  }

  custom_data = base64encode(<<-EOF
    #!/bin/bash
    set -euo pipefail
    apt-get update
    apt-get install -y ca-certificates curl gnupg
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable docker
    systemctl start docker
    usermod -aG docker ${var.admin_username}
  EOF
  )
}

output "instance_public_ip" {
  value = azurerm_public_ip.devbox.ip_address
}
`
