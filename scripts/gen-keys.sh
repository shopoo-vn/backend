#!/usr/bin/env bash
# Generate the RS256 keypair used by the Auth Service to sign JWTs.
# Other services only need keys/jwt_public.pem to verify tokens.
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/keys"
mkdir -p "$DIR"
openssl genpkey -algorithm RSA -out "$DIR/jwt_private.pem" -pkeyopt rsa_keygen_bits:2048
openssl rsa -in "$DIR/jwt_private.pem" -pubout -out "$DIR/jwt_public.pem"
echo "RS256 keypair written to $DIR"
