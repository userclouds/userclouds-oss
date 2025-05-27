terraform {
  required_providers {
    userclouds = {
      source  = "registry.terraform.io/userclouds/userclouds"
      version = ">= 0.1.0"
    }
  }
}

# Instantiate the provider, configuring via provider attributes
# or via environment variables
provider "userclouds" {
  tenant_url = "https://mytenant.tenant.userclouds.com"
  # Set USERCLOUDS_CLIENT_ID and USERCLOUDS_CLIENT_SECRET
  # environment variables to avoid hardcoding secrets.
  # USERCLOUDS_TENANT_URL may be set in place of tenant_url as
  # well.
}

# Create an example resource
resource "userclouds_userstore_column" "sample" {
  name       = "sample_column"
  type       = "string"
  index_type = "none"
  is_array   = false
}
