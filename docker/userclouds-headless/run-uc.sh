#!/usr/bin/env bash

set -euo pipefail

migrate --noPrompt --logfile=/userclouds/logs/migration.log rootdb rootdbstatus companyconfig tenantdb status
provision --logfile=/userclouds/logs/provision.log provision company /userclouds/provisioning/company.json
provision --logfile=/userclouds/logs/provision.log provision tenant /userclouds/provisioning/tenant_console.json
provision --logfile=/userclouds/logs/provision.log provision company /userclouds/provisioning/company_uc_container_dev.json
provision --logfile=/userclouds/logs/provision.log provision tenant /userclouds/provisioning/tenant_uc_container_dev.json
echo "Provision completed"

# Config for ucconfig tool which container runner will be ruinning, based on tenant_uc_container_dev.json
export USERCLOUDS_TENANT_URL=http://container-dev.tenant.test.userclouds.tools:3040
export USERCLOUDS_CLIENT_ID=container_dev_client_id
export USERCLOUDS_CLIENT_SECRET=container_dev_client_secret_fake_dont_use_outside_of_dev
containerrunner --headless "/customer/config/${UC_CONFIG_MANIFEST_FILE:-}"
