#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

# this function is passed as a callable param to validate_environment, but
# SC isn't that sophisticated, so we need to disable the warning
# shellcheck disable=SC2317
function print_usage() {
  echo "usage: tools/redis-shell.sh environment [region]"
  echo "  environment: prod | staging | dev"
  echo "  region: aws-us-west-2 | aws-us-east-1 (no need to specifiy for dev)"
}

UCENV="${1:-}"

source tools/check-env.sh
validate_environment "$UCENV" print_usage

RES=$(cloud_environment "$UCENV")
if [ "$RES" == "0" ]; then
  UC_REGION="${2:-aws-us-west-2}" # safe default
  echo "In $UCENV universe, setting default region to $UC_REGION..."
else
  UC_REGION="themoon"
fi

FILE="config/base_${UCENV}.yaml"
HOST=$(RG=$UC_REGION yq eval '.cache.redis_caches.[] | select (.region == strenv(RG)) | .host' "$FILE")
PORT=$(RG=$UC_REGION yq eval '.cache.redis_caches.[] | select (.region == strenv(RG)) | .port' "$FILE")
DBNAME=$(RG=$UC_REGION yq eval '.cache.redis_caches.[] | select (.region == strenv(RG)) | .dbname' "$FILE")
REDIS_USER=$(RG=$UC_REGION yq eval '.cache.redis_caches.[] | select (.region == strenv(RG)) | .username' "$FILE")
REDIS_PW_REF=$(RG=$UC_REGION yq eval '.cache.redis_caches.[] | select (.region == strenv(RG)) | .password' "$FILE")

if [ -n "$REDIS_PW_REF" ] && [ "$REDIS_PW_REF" != "null" ]; then
  # # ensure AWS env vars are set for this environment, since we need to resolve the secret from AWS
  UC_UNIVERSE=$UCENV tools/ensure-aws-auth.sh
  AWSPREFIX="aws://secrets/"
  if [[ $REDIS_PW_REF =~ $AWSPREFIX ]]; then
    SECRET_ID=${REDIS_PW_REF#"$AWSPREFIX"} # trim off the aws prefix
    REDIS_PW=$(aws secretsmanager get-secret-value --region=us-west-2 --secret-id "$SECRET_ID" | jq -r .SecretString)
  else
    echo "Unexpected password format: $REDIS_PW_REF"
    exit 1
  fi
else
  REDIS_PW=""
fi

echo "connecting to $UCENV redis at $HOST:$PORT"
redis-cli --version
if [ -z "$REDIS_PW" ]; then
  redis-cli -h "$HOST" -p "$PORT" -n "$DBNAME"
else
  set +e
  timeout 3 redis-cli --tls -h "$HOST" -p "$PORT" -n "$DBNAME" --user "$REDIS_USER" --pass "$REDIS_PW" ping
  if [[ $? -eq 124 ]]; then
    set -e
    echo "************************************************************************************************"
    echo "Can't connect to redis in cloud env. Make sure you are connected to UserCloud VPC using the VPN."
    echo "************************************************************************************************"
    exit 1
  else
    set -e
    redis-cli --tls -h "$HOST" -p "$PORT" -n "$DBNAME" --user "$REDIS_USER" --pass "$REDIS_PW"
  fi
fi
