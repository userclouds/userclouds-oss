#! /bin/bash

set -euo pipefail

# This script is used to generate a new VPN server certificate. See vpc.hcl for
# Terraform configuration that imports this cert.

USAGE="Usage: $0 <universe> [region] (defaults to us-west-2)"

if [ $# -lt 1 ]; then
  echo "$USAGE"
  exit 1
fi

# UNIVERSE is the AWS profile name, so beside prod,staging & debug it can also be "root" for the UC root AWS account
UNIVERSE="$1"

if [ $# -eq 1 ]; then
  REGION="us-west-2"
elif [ $# -eq 2 ]; then
  REGION="$2"
else
  echo "$USAGE"
  exit 1
fi

# https://unix.stackexchange.com/a/84980
TMPDIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'gencert')
echo "Generating in $TMPDIR"
cd "$TMPDIR"

echo "Generating VPN server certificate for $UNIVERSE in $REGION"
# Following instructions from
# https://docs.aws.amazon.com/vpn/latest/clientvpn-admin/mutual.html, without
# generating a client cert because we are using SSO auth
git clone --depth 1 --branch v3.2.1 https://github.com/OpenVPN/easy-rsa.git
cd easy-rsa/easyrsa3
./easyrsa init-pki
EASYRSA_BATCH=1 ./easyrsa build-ca nopass
SERVER_CERT_NAME="vpn-$UNIVERSE-$REGION"
EASYRSA_AUTO_SAN=1 EASYRSA_BATCH=1 ./easyrsa --no-pass build-server-full "$SERVER_CERT_NAME"
mkdir results
mv pki/ca.crt results/
mv "pki/issued/$SERVER_CERT_NAME.crt" results/server.crt
mv "pki/private/$SERVER_CERT_NAME.key" results/server.key
cd results

echo "Certificate info:"
openssl x509 -in server.crt -text -noout
echo "Importing certificate to AWS ACM"
CERT_ARN=$(aws --profile "$UNIVERSE" --region "$REGION" --no-cli-pager acm import-certificate \
  --certificate fileb://server.crt --private-key fileb://server.key --certificate-chain fileb://ca.crt \
  --tags="Key=Description,Value=Generated by $(basename "$0")" \
  --output json | jq -r '.CertificateArn')
echo "Imported certificate ARN: $CERT_ARN"
CERT_ID=$(echo "$CERT_ARN" | rev | cut -d'/' -f1 | rev)
echo "See cert in: https://$REGION.console.aws.amazon.com/acm/home?region=$REGION#/certificates/$CERT_ID"
rm -rf "$TMPDIR"
echo "Done!"
