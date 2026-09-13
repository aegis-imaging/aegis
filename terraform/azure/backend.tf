# Terraform state is stored in Azure Blob Storage.
#
# Before running terraform init, create the state storage account manually (once):
#
#   az group create --name aegis-tfstate --location eastus
#   az storage account create \
#     --name aegistfstate \
#     --resource-group aegis-tfstate \
#     --sku Standard_LRS \
#     --allow-blob-public-access false
#   az storage container create \
#     --name tfstate \
#     --account-name aegistfstate
#
# Then run: terraform init
#   (Azure CLI auth or ARM_* env vars supply credentials automatically)

terraform {
  backend "azurerm" {
    resource_group_name  = "aegis-tfstate"
    storage_account_name = "aegistfstate18d5c432"
    container_name       = "tfstate"
    key                  = "azure/prod/terraform.tfstate"
  }
}
