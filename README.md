# 🦞→🦐 claw-migrate

**One-command migration from [OpenClaw](https://github.com/openclaw/openclaw) to [PicoClaw](https://github.com/sipeed/picoclaw)**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Release](https://img.shields.io/github/v/release/arunbluez/claw-migrate)](https://github.com/arunbluez/claw-migrate/releases)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-zero-brightgreen)](go.mod)

An interactive CLI wizard that handles the entire OpenClaw → PicoClaw migration — backup, install, migrate, verify, and uninstall — in a single guided experience.

```
  ╔═══════════════════════════════════════════════════════════╗
  ║                                                           ║
  ║   🦞 → 🦐  claw-migrate                                  ║
  ║   OpenClaw → PicoClaw Migration Wizard                    ║
  ║                                                           ║
  ╚═══════════════════════════════════════════════════════════╝
```

## Why?

PicoClaw uses **99% less RAM** (<10MB vs >1GB), boots **400× faster**, and runs on **$10 hardware**. But migrating your carefully curated agent personality, memory, skills, API keys, and channel configs manually is tedious and error-prone.

`claw-migrate` does it all in one interactive session.

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

## Install

### Pre-built binary (recommended)

```bash
# macOS (Apple Silicon)
curl -L https://github.com/arunbluez/claw-migrate/releases/latest/download/claw-migrate-darwin-arm64.zip -o claw-migrate.zip
unzip claw-migrate.zip && chmod +x claw-migrate
sudo mv claw-migrate /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/arunbluez/claw-migrate/releases/latest/download/claw-migrate-darwin-amd64.zip -o claw-migrate.zip
unzip claw-migrate.zip && chmod +x claw-migrate
sudo mv claw-migrate /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/arunbluez/claw-migrate/releases/latest/download/claw-migrate-linux-amd64.tar.gz | tar xz
sudo mv claw-migrate /usr/local/bin/

# Linux (ARM64)
curl -L https://github.com/arunbluez/claw-migrate/releases/latest/download/claw-migrate-linux-arm64.tar.gz | tar xz
sudo mv claw-migrate /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/arunbluez/claw-migrate.git
cd claw-migrate
make build
sudo make install
```

> **Zero dependencies** — only requires Go 1.21+ to build. No external packages.

## Usage

### Full interactive wizard

```bash
claw-migrate
```

This walks you through all 6 phases with confirmations at each step:

1. **Detect** — Finds your OpenClaw installation, lists workspace files, providers, channels
2. **Backup** — Creates `~/openclaw-backup-YYYYMMDD-HHMMSS.tar.gz`
3. **Install** — Downloads PicoClaw binary or builds from source
4. **Migrate** — Copies workspace, converts config, maps API keys
5. **Verify** — Confirms everything transferred correctly
6. **Uninstall** — Removes OpenClaw (optional, with double confirmation)

### Dry run (see what would happen)

```bash
claw-migrate --dry-run
```

### Skip specific phases

```bash
claw-migrate --skip-install      # Already have PicoClaw installed
claw-migrate --skip-uninstall    # Keep OpenClaw for now
```

## How It Works

### Config Conversion

OpenClaw uses a flat TypeScript config with camelCase keys. PicoClaw uses structured Go JSON with snake_case. `claw-migrate` handles the translation:

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

### Safety Features

- **Full backup first** — tar.gz of entire `~/.openclaw/` before any changes
- **Backup verification** — integrity check on the backup archive
- **No silent overwrites** — existing PicoClaw files get `.bak` copies
- **Double confirmation** — uninstall requires explicit `y` (defaults to `N`)
- **Dry run mode** — preview everything without touching the filesystem
- **Rollback guide** — printed if anything fails

## Project Structure

```
claw-migrate/
├── main.go                          # Interactive wizard orchestrator
├── internal/
│   ├── ui/ui.go                     # Terminal UI (colors, prompts, progress)
│   ├── detect/detect.go             # Installation detection
│   ├── backup/backup.go             # Backup creation & verification
│   ├── install/install.go           # PicoClaw download & install
│   ├── config/config.go             # Config format conversion
│   ├── migrate/migrate.go           # Workspace file migration
│   └── uninstall/uninstall.go       # OpenClaw removal
├── Makefile                         # Build targets
├── .goreleaser.yaml                 # Release automation
├── go.mod                           # Go module (zero deps)
├── LICENSE                          # MIT
└── README.md                        # This file
```

## Contributing

PRs welcome! The codebase is intentionally small (~2K lines across 8 files) and has zero external dependencies.

### Areas that need help

- **Testing** — Add unit tests for config conversion edge cases
- **Windows support** — PowerShell equivalents for backup/uninstall commands
- **More providers** — Expand the provider mapping table for new LLM vendors
- **Interactive config editor** — Let users edit API keys inline during migration
- **PicoClaw's built-in migrate** — Better integration with `picoclaw migrate` command

### Development

```bash
git clone https://github.com/arunbluez/claw-migrate.git
cd claw-migrate
make build        # Build binary
make test         # Run tests
make build-all    # Cross-compile for all platforms
```

## Rollback

If anything goes wrong, restore your OpenClaw backup:

```bash
# Stop PicoClaw
pkill -f picoclaw

# Restore OpenClaw
cd ~ && tar -xzf openclaw-backup-*.tar.gz
npm install -g openclaw@latest
openclaw gateway
```

## Related

- [PicoClaw](https://github.com/sipeed/picoclaw) — Ultra-lightweight AI assistant in Go
- [OpenClaw](https://github.com/openclaw/openclaw) — Full-featured AI assistant in TypeScript
- [`picoclaw migrate`](https://github.com/sipeed/picoclaw/pull/33) — PicoClaw's built-in migration command

## License

MIT — see [LICENSE](LICENSE)

---

*Built with 🦐 by [Arunkumar](https://github.com/arunbluez) — contributions welcome!*
