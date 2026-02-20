# 🦞→🦐 claw-migrate

**One-command switch from [OpenClaw](https://github.com/openclaw/openclaw) to [PicoClaw](https://github.com/sipeed/picoclaw) — backup, install, migrate, verify, and uninstall in a single interactive session.**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Release](https://img.shields.io/github/v/release/arunbluez/claw-migrate)](https://github.com/arunbluez/claw-migrate/releases)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-zero-brightgreen)](go.mod)

```
  ╔═══════════════════════════════════════════════════════════╗
  ║                                                           ║
  ║   🦞 → 🦐  claw-migrate                                  ║
  ║   OpenClaw → PicoClaw Migration Wizard                    ║
  ║                                                           ║
  ╚═══════════════════════════════════════════════════════════╝

  PHASE 1  Detecting installations
  ──────────────────────────────────────────────────────
  ✓ OpenClaw                   ~/.openclaw
  ✓ Providers                  anthropic, openrouter
  ✓ Channels                   telegram
  ✓ Workspace files            6/6 present
  ✓ Memory                     MEMORY.md (142 lines)
  ✓ MCP Servers                filesystem, github

  ? Ready to begin migration? [Y/n]
```

## Why?

PicoClaw uses **99% less RAM** (<10MB vs >1GB), boots **400× faster**, and runs on **$10 hardware**. Switching to it means rebuilding your entire agent setup from scratch — copying workspace files, converting config formats, mapping API keys, setting up channels, recreating cron jobs, and eventually cleaning up the old install.

`claw-migrate` turns all of that into one guided session.

## Why not just `picoclaw migrate`?

PicoClaw ships with a [built-in `picoclaw migrate` command](https://github.com/sipeed/picoclaw/pull/33) (since v0.1.1). It's good at the **data transfer step** — copying workspace files and converting config keys. `claw-migrate` actually calls it under the hood when available.

The difference is scope. Here's what each tool covers:

| Step | `picoclaw migrate` | `claw-migrate` |
|------|:------------------:|:--------------:|
| Detect & audit OpenClaw setup | — | ✅ |
| Backup OpenClaw with integrity check | — | ✅ |
| Download & install PicoClaw | — | ✅ |
| Run `picoclaw onboard` | — | ✅ |
| Copy workspace files | ✅ | ✅ |
| Convert config (camelCase → snake_case) | ✅ | ✅ |
| Map providers to `model_list` format | — | ✅ |
| Flag unsupported channels (WhatsApp, Signal) | — | ✅ |
| Flag items needing manual attention (MCP, cron) | — | ✅ |
| Verify migration succeeded | — | ✅ |
| Uninstall OpenClaw (binary + data + launch agents) | — | ✅ |
| Dry-run mode | ✅ | ✅ |
| Rollback instructions | — | ✅ |

**TL;DR** — `picoclaw migrate` is the data transfer step. `claw-migrate` is the end-to-end onboarding experience for someone switching platforms. If you already have PicoClaw installed and just want to pull your workspace over, `picoclaw migrate` is all you need.

## What Gets Migrated

| Data | Status | Notes |
|------|--------|-------|
| Workspace files (SOUL.md, IDENTITY.md, AGENTS.md, etc.) | ✅ Auto | Direct copy — identical format |
| Long-term memory (`memory/`) | ✅ Auto | Direct copy |
| Custom skills (`skills/`) | ✅ Auto | Direct copy |
| Provider API keys | ✅ Auto | Converted to PicoClaw's `model_list` format |
| Channel configs (Telegram, Discord, Slack) | ✅ Auto | Token/credentials transferred |
| Heartbeat settings | ✅ Auto | Interval and tasks preserved |
| MCP connections | ⚠️ Semi | Migrated, but verify format manually |
| Cron jobs | ❌ Manual | Recreate with `picoclaw cron add` |
| Session history | ❌ Lost | Incompatible serialization format |

## Quick Start

Download and run — that's it. No `PATH` setup, no shell config, no `sudo`.

**macOS (Apple Silicon — M1/M2/M3/M4)**
```bash
curl -L https://github.com/arunbluez/claw-migrate/releases/latest/download/claw-migrate-macos-apple-silicon.zip -o claw-migrate.zip
unzip claw-migrate.zip
chmod +x claw-migrate
./claw-migrate
```

**macOS (Intel)**
```bash
curl -L https://github.com/arunbluez/claw-migrate/releases/latest/download/claw-migrate-macos-intel.zip -o claw-migrate.zip
unzip claw-migrate.zip
chmod +x claw-migrate
./claw-migrate
```

**Linux (x86_64)**
```bash
curl -L https://github.com/arunbluez/claw-migrate/releases/latest/download/claw-migrate-linux-x86_64.tar.gz | tar xz
chmod +x claw-migrate
./claw-migrate
```

**Linux (ARM64 / Raspberry Pi)**
```bash
curl -L https://github.com/arunbluez/claw-migrate/releases/latest/download/claw-migrate-linux-arm64.tar.gz | tar xz
chmod +x claw-migrate
./claw-migrate
```

> **Not sure which to pick?** Run `uname -m` — `arm64` or `aarch64` means ARM64, `x86_64` means amd64.

<details>
<summary><strong>Build from source</strong></summary>

Requires Go 1.21+. Zero external dependencies.

```bash
git clone https://github.com/arunbluez/claw-migrate.git
cd claw-migrate
make build
./bin/claw-migrate
```

</details>

## Usage

### Interactive mode (recommended)

```bash
./claw-migrate
```

Shows a menu:
```
? What would you like to do?
  1) Migrate — Full OpenClaw → PicoClaw migration
  2) Backup  — Create a backup of OpenClaw
  3) Restore — Restore OpenClaw from a backup
```

### Direct commands

```bash
./claw-migrate migrate     # Full 6-phase migration wizard
./claw-migrate backup      # Just backup ~/.openclaw/
./claw-migrate restore     # Restore from a previous backup
```

### Migration phases

The `migrate` command walks you through 6 phases, with confirmations at each step:

1. **Detect** — Scans for OpenClaw & PicoClaw, audits workspace files, providers, channels, MCP servers
2. **Backup** — Creates `~/openclaw-backup-YYYYMMDD-HHMMSS.tar.gz` with integrity verification
3. **Install** — Downloads PicoClaw binary (or builds from source), runs `picoclaw onboard`
4. **Migrate** — Copies entire workspace, converts config, checks model version
5. **Verify** — Confirms everything transferred, prints test commands to try
6. **Uninstall** — Removes OpenClaw binary, data, and macOS launch agents (optional, double confirmation)

### Dry run

```bash
claw-migrate --dry-run
```

Preview every action without touching the filesystem.

### Skip specific phases

```bash
claw-migrate --skip-install      # Already have PicoClaw installed
claw-migrate --skip-uninstall    # Keep OpenClaw around for now
```

## How It Works

### Config Conversion

OpenClaw uses a flat TypeScript config with camelCase keys. PicoClaw uses structured Go JSON with snake_case and a new `model_list` format. `claw-migrate` handles the translation automatically:

**OpenClaw** (`~/.openclaw/openclaw.json`):
```json
{
  "providers": {
    "anthropic": { "apiKey": "sk-ant-..." },
    "openrouter": { "apiKey": "sk-or-..." }
  },
  "agent": { "model": "anthropic/claude-sonnet-4-5", "maxTokens": 8192 },
  "channels": { "telegram": { "enabled": true, "token": "123:ABC" } }
}
```

**PicoClaw** (`~/.picoclaw/config.json`) — auto-generated:
```json
{
  "model_list": [
    { "model_name": "anthropic", "model": "anthropic/claude-sonnet-4.6", "api_key": "sk-ant-..." },
    { "model_name": "openrouter", "model": "openrouter/anthropic/claude-sonnet-4.6", "api_key": "sk-or-..." }
  ],
  "agents": { "defaults": { "model": "anthropic", "max_tokens": 8192 } },
  "channels": { "telegram": { "enabled": true, "token": "123:ABC" } },
  "heartbeat": { "enabled": true, "interval": 30 }
}
```

### Safety

- **Full backup first** — `tar.gz` of entire `~/.openclaw/` before any changes
- **Backup verification** — integrity check on the archive
- **No silent overwrites** — existing PicoClaw files get `.bak` copies
- **Double confirmation** — uninstall defaults to `N`, requires explicit `y`
- **Dry run mode** — preview everything without touching the filesystem
- **Rollback instructions** — printed if anything fails

## Project Structure

```
claw-migrate/
├── main.go                          # Interactive wizard (6 phases)
├── internal/
│   ├── ui/ui.go                     # Terminal UI (colors, prompts, progress)
│   ├── detect/detect.go             # Find & audit OpenClaw/PicoClaw installs
│   ├── backup/backup.go             # Backup creation & verification
│   ├── install/install.go           # PicoClaw download & install
│   ├── config/config.go             # Config format conversion
│   ├── migrate/migrate.go           # Workspace file migration
│   └── uninstall/uninstall.go       # OpenClaw removal & cleanup
├── Makefile                         # Build targets
├── .goreleaser.yaml                 # Release automation
├── go.mod                           # Go module (zero deps)
├── LICENSE                          # MIT
└── README.md
```

## Contributing

PRs welcome! The codebase is ~2K lines across 8 files with zero external dependencies.

### Areas that need help

- **Testing** — Unit tests for config conversion edge cases
- **Windows support** — PowerShell equivalents for backup/uninstall
- **More providers** — Expand the provider mapping table for new LLM vendors
- **Interactive config editor** — Edit API keys inline during migration
- **Session history converter** — TypeScript JSON → Go format (hard problem)

### Development

```bash
git clone https://github.com/arunbluez/claw-migrate.git
cd claw-migrate
make build        # Build binary
make test         # Run tests
make build-all    # Cross-compile for all platforms
```

## Rollback

If anything goes wrong, restore from the backup created in Phase 2:

```bash
./claw-migrate restore
```

Or manually:

```bash
pkill -f picoclaw
cd ~ && tar -xzf openclaw-backup-*.tar.gz
npm install -g openclaw@latest
openclaw gateway
```

## Related

- [PicoClaw](https://github.com/sipeed/picoclaw) — Ultra-lightweight AI assistant in Go
- [OpenClaw](https://github.com/openclaw/openclaw) — Full-featured AI assistant in TypeScript
- [`picoclaw migrate`](https://github.com/sipeed/picoclaw/pull/33) — PicoClaw's built-in data transfer command

## License

MIT — see [LICENSE](LICENSE)

---

*Built with 🦐 by [Arunkumar](https://github.com/arunbluez) — contributions welcome!*