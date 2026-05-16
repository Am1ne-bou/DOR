# Contributors

DOR started as a 7-person school project at ENSEIRB-MATMECA (S8 Telecom, 2025-2026).
I'm maintaining this fork because I genuinely like what the project is trying to do --
a lightweight, config-free anonymous relay network anyone can join from a laptop or VM.
This fork is where I'm taking it further.

## Protocol & Network Core

**BOUSSENNA Mohamed Amine**
- Initial multi-hop P2P relay prototype (Jan 2026)
- Directory server design and node registration protocol
- SQLite persistence integration
- OnionLayer pipeline: RSA-2048 key exchange + AES-256-GCM per-hop encryption
- NACK/retry system with per-hop random identifiers
- Broadcast encryption and anonymous path selection (`Broadcast.go`)
- SSEND/SBENCH protocol and multi-candidate forwarding (`super_send.go`)
- Automated benchmark and latency simulation suite
- Security fixes: unchecked reads, public key map race, TLS key purge from history
- CI pipeline, package restructure, structured logging (fork maintenance)

## Web Interface & Dashboard

- Node web UI for local send/receive
- Real-time dashboard with node telemetry
- Public-facing frontend demo

## Infrastructure & Deployment

- Docker images for amd64/arm with hardware-profile simulation
- Automation scripts for Linux/macOS/Windows
- Environment variable wiring, UUID generation, random port assignment

## Testing

- Unit tests for Node and OnionLayer components
- Private key regeneration on startup
