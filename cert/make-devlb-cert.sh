#!/bin/bash

CERT_DIR=$(dirname "$(readlink -f "$0")")
DOMAIN=dev.userclouds.tools

# Generate a root CA for signing the devlb certificate. It would be great if we
# could just generate and trust the devlb certificate. Frustratingly, Firefox
# only allows importing CAs (not leaf certificates).
openssl req -x509 -newkey rsa:4096 -sha256 -days 730 \
  -nodes -keyout "$CERT_DIR/devlb-CA.key" -out "$CERT_DIR/devlb-CA.crt" \
  -subj "/CN=UserClouds devlb CA" \
  -addext "subjectAltName=DNS:$DOMAIN,DNS:*.$DOMAIN,DNS:*.tenant.$DOMAIN"
# ^ unsure if SAN works for restricting the authority of root CAs, but seems like there is no downside

# Generate devlb leaf certificate
openssl req -x509 -newkey rsa:4096 -sha256 -days 730 \
  -CA "$CERT_DIR/devlb-CA.crt" -CAkey "$CERT_DIR/devlb-CA.key" \
  -nodes -keyout "$CERT_DIR/devlb.key" -out "$CERT_DIR/devlb.crt" \
  -subj "/CN=UserClouds devlb" \
  -addext "subjectAltName=DNS:$DOMAIN,DNS:*.$DOMAIN,DNS:*.tenant.$DOMAIN" \
  -addext "basicConstraints=critical,CA:false"

# Delete the root CA key so it can't be used for anything else
rm "$CERT_DIR/devlb-CA.key"
