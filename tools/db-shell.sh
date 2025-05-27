#!/usr/bin/env bash

# TODO this whole thing is janky but it works for now, we can switch it up
# once we use a credential manager for the prod DB creds etc

set -euo pipefail
IFS=$'\n\t'

# this function is passed as a callable param to validate_environment, but
# SC isn't that sophisticated, so we need to disable the warning
# shellcheck disable=SC2317
function print_usage() {
  echo "usage: tools/db-shell.sh environment"
  echo "  environment: prod | staging | dev"
}

UCENV="${1:-}"

source tools/check-env.sh
validate_environment "$UCENV" print_usage

# ensure AWS env vars are set for this environment
UC_UNIVERSE=$UCENV tools/ensure-aws-auth.sh

PS3='Connect to DB for which service? [1-8] '
OPTIONS=("companyconfig" "rootdb" "rootdbstatus" "logserver" "tenantdb" "console tenant" "search companies" "search tenants")
select OPT in "${OPTIONS[@]}"; do
  case $OPT in
  "companyconfig" | "rootdb" | "rootdbstatus" | "logserver")
    if ! DB_URL=$(go run ./cmd/psqldbshell "$OPT" | tail -n1); then
      echo "Failed to get database URL for $OPT"
      exit 1
    fi
    break
    ;;
  "tenantdb")
    make bin/tenantdbshell
    UC_UNIVERSE="$UCENV" bin/tenantdbshell --prompt
    exit 0
    ;;
  "console tenant")
    make bin/tenantdbshell
    if [[ ${UCENV} == "dev" ]]; then
      CONSOLE_TENANT_ID=$(yq eval .console_tenant_id "config/console/dev.yaml")
    else
      CONSOLE_TENANT_ID=$(yq eval .config.common.console_tenant_id "helm/userclouds/values-$UCENV.yaml")
    fi
    echo "Connecting to console tenant $CONSOLE_TENANT_ID for $UCENV"
    UC_UNIVERSE="$UCENV" bin/tenantdbshell "$CONSOLE_TENANT_ID"
    exit 0
    ;;
  "search companies")
    make bin/tenantdbshell
    UC_UNIVERSE="$UCENV" bin/tenantdbshell --prompt --search --companies
    exit 0
    ;;
  "search tenants")
    make bin/tenantdbshell
    UC_UNIVERSE="$UCENV" bin/tenantdbshell --prompt --search
    exit 0
    ;;
  *)
    echo "Invalid service"
    exit 1
    ;;
  esac
done
echo ""

if [[ -n ${DB_URL:-} ]]; then
  psql "$DB_URL"
else
  # resolve the secret if needed
  AWSPREFIX="aws://secrets/"
  DEVPREFIX="dev-literal://"
  if [[ $PASS =~ $AWSPREFIX ]]; then
    PASS=${PASS#"$AWSPREFIX"} # trim off the aws prefix
    PASS=$(aws secretsmanager get-secret-value --region=us-west-2 --secret-id "$PASS" | jq -r .SecretString)
    if [[ $PASS =~ "{" ]]; then
      PASS=$(echo "$PASS" | jq -r .string)
    fi
  elif [[ $PASS =~ $DEVPREFIX ]]; then
    PASS=${PASS#"$DEVPREFIX"} # trim off the dev-literal prefix and that's it
  fi

  if [ "$PASS" == "null" ]; then
    PASS=""
  fi
  PGPASSWORD="$PASS" psql -h "$HOST" -p "$PORT" -U "$USR" "$DBNAME"
fi
