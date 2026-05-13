#!/bin/bash
# Generates a self-signed TLS certificate for the directory server.
# Run this once before building. Never commit the output files.
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CERT_DIR="$ROOT/node_server/List_Serveur"
EMBED_DIR="$ROOT/node_server/model"

openssl req -x509 -newkey rsa:2048 -keyout "$CERT_DIR/key.pem" \
  -out "$CERT_DIR/cert.pem" -days 365 -nodes \
  -subj "/CN=dor-directory/O=DOR/C=FR" \
  -addext "subjectAltName=IP:127.0.0.1,IP:0.0.0.0"

cp "$CERT_DIR/cert.pem" "$EMBED_DIR/cert.pem"

echo "Certificate written to:"
echo "  $CERT_DIR/cert.pem"
echo "  $CERT_DIR/key.pem"
echo "  $EMBED_DIR/cert.pem  (embed copy)"
