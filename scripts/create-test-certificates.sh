#!/bin/bash

# Disable Git Bash path conversion on Windows
export MSYS_NO_PATHCONV=1

# Script dir (so it works when run from another dir)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# cd to fixture dir
CERTS_DIR="$SCRIPT_DIR/../fixtures/certs"
mkdir -p "$CERTS_DIR"

# Create CA key and cert
cd "$CERTS_DIR"

# Create CA key and cert
openssl req -x509 -newkey rsa:2048 -keyout ca.key -out ca.crt -days 3650 -nodes -subj "//CN=test-ca"

# Create client key and cert
openssl req -newkey rsa:2048 -keyout client.key -out client.csr -days 3650 -nodes -subj "//CN=test-client"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 3650

# Clean up CSR file
rm -f client.csr ca.srl

echo "Test certificates created successfully in $CERTS_DIR"