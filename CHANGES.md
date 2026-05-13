# CHANGES - Repo cleanup (2026-05-13)

## 1. Sensitive files purged from git history

**What:** `node_server/List_Serveur/cert.pem`, `node_server/List_Serveur/key.pem`,
`node_server/model/cert.pem` removed from every commit in history using `git filter-repo`.

**Why:** A team member committed the TLS private key and self-signed certificate directly
into the repo (commit message even acknowledged it). Private keys in git history are
permanently compromised - anyone who ever cloned the repo has them. The files existed in
one commit (`5c07611`) and propagated through all subsequent commits.

**What you must do after this:**
- `git push origin --force --all` to overwrite the remote history.
- Regenerate the certificate: `bash scripts/gen-cert.sh` (old cert is burned).
- Warn any other clones/forks - their histories still contain the key.

`*.pem` and `*.key` are in `.gitignore` and will never be auto-staged again.

---

## 2. Old documentation deleted

**What:** Removed 9 stale `.md` files and the `wiki/` directory:
- `Gestion Branches.md` - internal git notes, not useful to readers
- `README-Localhost-to-IP.md` - setup notes superseded by Docker
- `README-Docker.md` - merged into new README
- `node_server/Readme nack.md` - branch-specific notes from feature/nack
- `node_server/EXPLICATION.md` - early prototype notes, outdated
- `wiki/ExplicationServeur.md`, `wiki/SQLManager.md`, `wiki/ExplicationScoring.md`,
  `wiki/ExplicationAck.md` - per-feature wikis written during development, now stale

**Why:** Stale docs are worse than no docs. They described code paths that have since
changed, referenced branch names that no longer exist, and made the repo look unfinished.
A recruiter or contributor reading them would get wrong information.

**Replaced by:** A single `README.md` covering architecture, quick start, env vars, and
protocol. A `SECURITY.md` covering threat model and known limitations.

---

## 3. README rewritten

**What:** `README.md` rewritten from scratch (old: 25 lines, placeholder text).

**Why:** The old README said "Projet S8 DOR" and nothing else. The new one covers:
architecture, Docker quick start, local quick start, environment variables, wire protocol,
and a pointer to SECURITY.md.

---

## 4. SECURITY.md added

**What:** New `SECURITY.md` documenting threat model, crypto primitives used,
known limitations, and how to generate the TLS certificate.

**Why:** Explicitly documenting what the system does and does not protect against is
standard practice. It also explains the intentional shortcuts (InsecureSkipVerify,
no web UI auth) so they don't look like accidents to a reviewer.

---

## 5. Scripts moved to `scripts/`

**What:** Six shell scripts moved from repo root to `scripts/`:
- `st.sh` -> `scripts/st.sh`
- `start_node.sh` -> `scripts/start_node.sh`
- `start_nodes_tmux.sh` -> `scripts/start_nodes_tmux.sh`
- `start-dor.sh` -> `scripts/start-dor.sh`
- `start-dor-mac.sh` -> `scripts/start-dor-mac.sh`
- `start-dor.ps1` -> `scripts/start-dor.ps1`

Internal paths fixed: scripts that used `cd "$(dirname "$0")"` updated to
`cd "$(dirname "$0")/.."` so they still resolve relative to repo root.

**Why:** Loose scripts in a repo root signal a messy project. `scripts/` is the
conventional location; it keeps the root clean and makes it obvious these are
dev helpers, not build artifacts.

---

## 6. scripts/gen-cert.sh added

**What:** New script that generates a self-signed cert + key and places them in the
correct locations (`node_server/List_Serveur/` and `node_server/model/` for the
`go:embed` directive).

**Why:** The cert was previously committed because there was no documented way to
regenerate it. Now there is. Any developer can run `bash scripts/gen-cert.sh` once
before building.

---

## What still needs doing (remote)

```bash
# Re-add the remote (filter-repo removes it as a safety measure)
git remote add origin https://github.com/Am1ne-bou/DOR.git

# Force-push the rewritten history
git push origin --force --all
```

After pushing, GitHub will show the pem/key files as gone from all commits.
