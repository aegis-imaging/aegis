# AEGIS — DIMSE Receiver VM
#
# Azure Container Apps cannot expose raw TCP port 11112 needed by DICOM C-STORE SCP.
# The dimse-receiver runs on an Azure Linux Virtual Machine.
#
# All DIMSE resources are only created when var.dimse_receiver_image is non-empty.
# Set it in terraform.tfvars when you're ready to deploy the DIMSE receiver.
#
# On each image update: run a VM extension script or Azure Run Command to pull
# the new image and restart the container (same pattern as the GCP Compute Engine
# and AWS EC2 deployments).

locals {
  dimse_enabled = var.dimse_receiver_image != ""
}

resource "azurerm_subnet" "dimse" {
  count                = local.dimse_enabled ? 1 : 0
  name                 = "dimse"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.4.0/24"]
}

resource "azurerm_public_ip" "dimse" {
  count               = local.dimse_enabled ? 1 : 0
  name                = "${local.prefix}-dimse-ip"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.tags
}

resource "azurerm_network_security_group" "dimse" {
  count               = local.dimse_enabled ? 1 : 0
  name                = "${local.prefix}-dimse-nsg"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  security_rule {
    name                         = "allow-dicom-inbound"
    priority                     = 100
    direction                    = "Inbound"
    access                       = "Allow"
    protocol                     = "Tcp"
    source_port_range            = "*"
    destination_port_range       = "11112"
    source_address_prefixes      = var.dimse_source_ranges
    destination_address_prefix   = "*"
  }

  security_rule {
    name                       = "allow-ssh-inbound"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  tags = local.tags
}

resource "azurerm_network_interface" "dimse" {
  count               = local.dimse_enabled ? 1 : 0
  name                = "${local.prefix}-dimse-nic"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  ip_configuration {
    name                          = "dimse-ip-config"
    subnet_id                     = azurerm_subnet.dimse[0].id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.dimse[0].id
  }

  tags = local.tags
}

resource "azurerm_network_interface_security_group_association" "dimse" {
  count                     = local.dimse_enabled ? 1 : 0
  network_interface_id      = azurerm_network_interface.dimse[0].id
  network_security_group_id = azurerm_network_security_group.dimse[0].id
}

resource "azurerm_managed_disk" "dimse_data" {
  count                = local.dimse_enabled ? 1 : 0
  name                 = "${local.prefix}-dimse-data"
  location             = azurerm_resource_group.main.location
  resource_group_name  = azurerm_resource_group.main.name
  storage_account_type = "Standard_LRS"
  create_option        = "Empty"
  disk_size_gb         = 32
  tags                 = local.tags
}

resource "azurerm_user_assigned_identity" "dimse" {
  count               = local.dimse_enabled ? 1 : 0
  name                = "${local.prefix}-dimse"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags
}

resource "azurerm_role_assignment" "dimse_acr_pull" {
  count                = local.dimse_enabled ? 1 : 0
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.dimse[0].principal_id
}

resource "azurerm_role_assignment" "dimse_blob" {
  count                = local.dimse_enabled ? 1 : 0
  scope                = azurerm_storage_account.dicom.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.dimse[0].principal_id
}

resource "azurerm_linux_virtual_machine" "dimse" {
  count               = local.dimse_enabled ? 1 : 0
  name                = "${local.prefix}-dimse-receiver"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  size                = var.dimse_vm_size
  admin_username      = "aegis"

  network_interface_ids = [azurerm_network_interface.dimse[0].id]

  # Use managed identity for ACR pull — no stored credentials
  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.dimse[0].id]
  }

  admin_ssh_key {
    username   = "aegis"
    public_key = var.dimse_ssh_public_key
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
    disk_size_gb         = 30
  }

  source_image_reference {
    publisher = "Debian"
    offer     = "debian-12"
    sku       = "12"
    version   = "latest"
  }

  # Startup script: log into ACR via managed identity, pull image, run container.
  # Image tag is passed via custom_data to allow Cloud Build-style image updates.
  custom_data = base64encode(<<-EOF
    #!/bin/bash
    set -e

    # Install Docker
    apt-get update -qq
    apt-get install -y docker.io azure-cli

    # Log into ACR using VM managed identity
    ACR_TOKEN=$(az acr login --name ${azurerm_container_registry.main.name} --expose-token --output tsv --query accessToken)
    docker login ${local.acr_server} --username 00000000-0000-0000-0000-000000000000 --password "$ACR_TOKEN"

    # Pull and start the DIMSE receiver
    DIMSE_IMAGE="${var.dimse_receiver_image}"
    docker pull "$DIMSE_IMAGE"
    docker rm -f dimse-receiver 2>/dev/null || true
    docker run -d \
      --name dimse-receiver \
      --restart unless-stopped \
      -p 11112:11112 \
      -e DIMSE_AE_TITLE=AEGIS \
      -e DIMSE_PORT=11112 \
      -e STORAGE_MODE=azure \
      -e AZURE_STORAGE_ACCOUNT=${azurerm_storage_account.dicom.name} \
      -e AZURE_STORAGE_CONTAINER=dicom \
      -e API_URL=https://${azurerm_container_app.api.ingress[0].fqdn} \
      -e DIMSE_PROJECT_SLUG=default \
      "$DIMSE_IMAGE"
  EOF
  )

  tags = local.tags

  lifecycle {
    # Cloud Build updates the running container via az vm run-command — don't
    # let terraform revert those in-flight image changes on next apply.
    ignore_changes = [custom_data]
  }
}

resource "azurerm_virtual_machine_data_disk_attachment" "dimse" {
  count              = local.dimse_enabled ? 1 : 0
  managed_disk_id    = azurerm_managed_disk.dimse_data[0].id
  virtual_machine_id = azurerm_linux_virtual_machine.dimse[0].id
  lun                = 0
  caching            = "ReadWrite"
}
