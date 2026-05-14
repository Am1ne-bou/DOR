# Security

## Threat model

DOR is a proof-of-concept onion routing system built for academic purposes. The goal is to hide the source and destination of a message from any single relay node in the circuit.

**What it protects against:**
- A single compromised relay knowing both sender and recipient
- Passive eavesdropping on a single network link (AES-256-GCM per hop)
- Key reuse attacks (ephemeral AES key per message, RSA-OAEP wrapping)

**What it does NOT protect against:**
- Global passive adversary (traffic correlation)
- Timing attacks across hops
- A compromised directory server (single point of trust)

## Cryptography

| Primitive | Usage |
|---|---|
| RSA-2048 OAEP (SHA-256) | Per-hop AES key wrapping |
| AES-256-GCM | Payload encryption per layer |
| TLS (self-signed) | Node <-> directory server channel |

## Known limitations (intentional for school context)

- `InsecureSkipVerify: true` in TLS dial - the cert is pinned via embedded PEM, not verified against a CA. Acceptable for a closed demo network; not for production.
- The directory server is a single point of failure and trust. No distributed DHT.
- The web UI (`ENABLE_WEB=1`) binds on `0.0.0.0:9090` with no authentication. Only enable it in a trusted environment.
- `publicKeys` cache in the node has no mutex - known race under concurrent load.

## Generating a TLS certificate

The directory server requires a self-signed cert. Never commit `cert.pem` or `key.pem`.

```bash
bash scripts/gen-cert.sh
# produces: node_server/List_Serveur/cert.pem
#           node_server/List_Serveur/key.pem
#           node_server/model/cert.pem  (copy for embed)
```

The cert is embedded at compile time via `//go:embed cert.pem` in `Node.go`. Regenerate and recompile after any cert rotation.

## Reporting issues

This is a personal academic fork. Open a GitHub issue or contact me directly.
