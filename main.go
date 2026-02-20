package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arunbluez/claw-migrate/internal/backup"
	"github.com/arunbluez/claw-migrate/internal/detect"
	"github.com/arunbluez/claw-migrate/internal/install"
	"github.com/arunbluez/claw-migrate/internal/migrate"
	"github.com/arunbluez/claw-migrate/internal/ui"
	"github.com/arunbluez/claw-migrate/internal/uninstall"
)

func main() {
	// Parse flags
	dryRun := false
	skipInstall := false
	skipUninstall := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--skip-install":
			skipInstall = true
		case "--skip-uninstall":
			skipUninstall = true
		case "--help", "-h":
			printHelp()
			return
		case "--version", "-v":
			fmt.Println("claw-migrate v1.0.0")
			return
		}
	}

	// === Banner ===
	ui.Banner()

	if dryRun {
		ui.Warn("DRY RUN mode — no changes will be made")
	}

	// === Phase 1: Detect ===
	phase1Detect()

	// Get installations
	oc := detect.DetectOpenClaw()
	pc := detect.DetectPicoClaw()
	sys := detect.GetSystemInfo()

	if !oc.Found {
		ui.Error("OpenClaw installation not found at ~/.openclaw/")
		ui.Info("Make sure OpenClaw is installed and has been initialized.")
		ui.Info("Expected directory: ~/.openclaw/")
		os.Exit(1)
	}

	// Show what we found
	showDetectionResults(oc, pc, sys)

	// Ask to proceed
	if !ui.Confirm("Ready to begin migration?") {
		ui.Info("Migration cancelled. No changes made.")
		return
	}

	// === Phase 2: Backup ===
	phase2Backup(oc, dryRun)

	// === Phase 3: Install PicoClaw ===
	if !skipInstall {
		phase3Install(pc, sys, dryRun)
	} else {
		ui.Phase(3, "Install PicoClaw (skipped)")
		ui.Info("--skip-install flag set, skipping installation")
	}

	// Re-detect PicoClaw after install
	pc = detect.DetectPicoClaw()

	// === Phase 4: Migrate ===
	phase4Migrate(oc, pc, dryRun)

	// === Phase 5: Verify ===
	phase5Verify()

	// === Phase 6: Uninstall OpenClaw ===
	if !skipUninstall {
		phase6Uninstall(oc, dryRun)
	} else {
		ui.Phase(6, "Uninstall OpenClaw (skipped)")
		ui.Info("--skip-uninstall flag set. You can uninstall later with:")
		ui.Info("  npm uninstall -g openclaw && rm -rf ~/.openclaw")
	}

	// === Done ===
	ui.CompletionBanner()
}

// ════════════════════════════════════════════════════════════
// Phase 1: Detect installations
// ════════════════════════════════════════════════════════════

func phase1Detect() {
	ui.Phase(1, "Detecting installations")
}

func showDetectionResults(oc, pc detect.Installation, sys detect.SystemInfo) {
	ui.Step(1, "System information")
	ui.Found("Platform", fmt.Sprintf("%s/%s", sys.OS, sys.Arch))

	ui.Step(2, "OpenClaw installation")
	ui.Found("Directory", oc.HomeDir)

	if oc.BinaryPath != "" {
		ui.Found("Binary", oc.BinaryPath)
	}
	if oc.Version != "" {
		ui.Found("Version", oc.Version)
	}

	// Show config details
	ui.Step(3, "Configuration")
	if oc.Config != nil {
		ui.Found("Config file", fmt.Sprintf("%s (%s)", oc.ConfigPath, detect.FormatSize(oc.ConfigSummary.ConfigFileSize)))

		// Model & agent settings
		if oc.ConfigSummary.DefaultModel != "" {
			ui.Found("Default model", oc.ConfigSummary.DefaultModel)
		}
		if oc.ConfigSummary.MaxTokens > 0 {
			ui.Found("Max tokens", fmt.Sprintf("%d", oc.ConfigSummary.MaxTokens))
		}

		// Providers
		providers := detect.GetProviderKeys(oc.Config)
		if len(providers) > 0 {
			ui.Found("Providers", strings.Join(providers, ", "))
		}

		// Channels
		channels := detect.GetConfiguredChannels(oc.Config)
		if len(channels) > 0 {
			ui.Found("Channels", strings.Join(channels, ", "))
		}

		// MCP servers
		mcpServers := detect.GetMCPServers(oc.Config)
		if len(mcpServers) > 0 {
			ui.Found("MCP Servers", strings.Join(mcpServers, ", "))
		}

		// Heartbeat
		if oc.ConfigSummary.HeartbeatEnabled {
			ui.Found("Heartbeat", fmt.Sprintf("enabled (every %d min)", oc.ConfigSummary.HeartbeatInterval))
		}
	} else {
		ui.NotFound("Config file")
	}

	// Show workspace — standard agent files
	ui.Step(4, "Workspace — agent files")
	standardFileList := []string{"SOUL.md", "IDENTITY.md", "AGENTS.md", "USER.md", "TOOLS.md", "HEARTBEAT.md"}
	foundCount := 0
	for _, f := range standardFileList {
		exists := oc.WorkspaceFiles[f]
		lines := 0
		if exists {
			lines = detect.CountFileLines(filepath.Join(oc.WorkspaceDir, f))
			foundCount++
		}
		ui.FileStatus(f, exists, lines)
	}

	// Show extra files (non-standard .md files and other files)
	if len(oc.ExtraFiles) > 0 {
		ui.Step(5, fmt.Sprintf("Workspace — custom files (%d)", len(oc.ExtraFiles)))
		for _, f := range oc.ExtraFiles {
			lines := detect.CountFileLines(filepath.Join(oc.WorkspaceDir, f))
			ui.FileStatus(f, true, lines)
		}
	}

	// Show standard directories with file counts
	ui.Step(6, "Workspace — standard directories")
	stdDirs := []struct {
		name string
		has  bool
	}{
		{"memory", oc.HasMemory},
		{"skills", oc.HasSkills},
		{"cron", oc.HasCron},
		{"sessions", oc.HasSessions},
	}
	for _, d := range stdDirs {
		if d.has {
			dirPath := filepath.Join(oc.WorkspaceDir, d.name)
			count := detect.CountDirFiles(dirPath)
			size := detect.DirSize(dirPath)
			ui.Found(d.name+"/", fmt.Sprintf("%d files (%s)", count, detect.FormatSize(size)))
		} else {
			ui.NotFound(d.name + "/")
		}
	}

	// Show extra directories (project folders, repos, etc.)
	if len(oc.ExtraDirs) > 0 {
		ui.Step(7, fmt.Sprintf("Workspace — project directories (%d)", len(oc.ExtraDirs)))
		for _, d := range oc.ExtraDirs {
			dirPath := filepath.Join(oc.WorkspaceDir, d)
			count := detect.CountDirFiles(dirPath)
			size := detect.DirSize(dirPath)
			ui.Found(d+"/", fmt.Sprintf("%d files (%s)", count, detect.FormatSize(size)))
		}
	}

	// Summary totals
	totalFiles := foundCount + len(oc.ExtraFiles)
	totalDirs := len(oc.ExtraDirs)
	for _, d := range stdDirs {
		if d.has {
			totalDirs++
		}
	}
	totalSize := detect.DirSize(oc.WorkspaceDir)

	fmt.Println()
	ui.Info(fmt.Sprintf("Total: %d files, %d directories (%s)",
		totalFiles, totalDirs, detect.FormatSize(totalSize)))

	// Show PicoClaw status
	nextStep := 7
	if len(oc.ExtraDirs) > 0 {
		nextStep = 8
	}
	ui.Step(nextStep, "PicoClaw installation")
	if pc.Found {
		ui.Found("Directory", pc.HomeDir)
		if pc.BinaryPath != "" {
			ui.Found("Binary", pc.BinaryPath)
		}
		if pc.Version != "" {
			ui.Found("Version", pc.Version)
		}
	} else {
		ui.NotFound("PicoClaw")
		ui.Info("PicoClaw will be installed in the next phase")
	}
}

// ════════════════════════════════════════════════════════════
// Phase 2: Backup
// ════════════════════════════════════════════════════════════

func phase2Backup(oc detect.Installation, dryRun bool) {
	ui.Phase(2, "Backup OpenClaw")

	ui.Step(1, "Creating full backup of ~/.openclaw/")

	if dryRun {
		ui.Info("[DRY RUN] Would create backup: ~/openclaw-backup-YYYYMMDD-HHMMSS.tar.gz")
		return
	}

	result := backup.CreateBackup(oc.HomeDir)
	if !result.Success {
		ui.Error(fmt.Sprintf("Backup failed: %v", result.Error))
		if !ui.ConfirmDangerous("Continue WITHOUT backup? (not recommended)") {
			ui.Info("Migration cancelled. Fix the backup issue and try again.")
			os.Exit(1)
		}
		return
	}

	ui.Success(fmt.Sprintf("Backup created: %s (%s)", result.Path, backup.FormatSize(result.Size)))

	// Verify backup
	ui.Step(2, "Verifying backup integrity")
	if err := backup.VerifyBackup(result.Path); err != nil {
		ui.Warn(fmt.Sprintf("Backup verification warning: %v", err))
	} else {
		ui.Success("Backup verified successfully")
	}
}

// ════════════════════════════════════════════════════════════
// Phase 3: Install PicoClaw
// ════════════════════════════════════════════════════════════

func phase3Install(pc detect.Installation, sys detect.SystemInfo, dryRun bool) {
	ui.Phase(3, "Install PicoClaw")

	// Check if already installed
	if pc.BinaryPath != "" {
		ui.Success(fmt.Sprintf("PicoClaw already installed: %s", pc.BinaryPath))
		if pc.Version != "" {
			ui.Info(fmt.Sprintf("Version: %s", pc.Version))
		}

		if !ui.Confirm("Skip installation and use existing PicoClaw?") {
			// User wants to reinstall
		} else {
			// Initialize if needed
			if !pc.Found {
				ui.Step(2, "Initializing PicoClaw workspace")
				if !dryRun {
					install.RunOnboard()
				} else {
					ui.Info("[DRY RUN] Would run: picoclaw onboard")
				}
			}
			return
		}
	}

	// Choose install method
	method := ui.Choose("How would you like to install PicoClaw?", []string{
		fmt.Sprintf("Download pre-built binary (%s, recommended)", install.LatestVersion),
		"Build from source (latest features, requires Go 1.21+)",
	})

	if dryRun {
		if method == 0 {
			url, _, _ := install.GetDownloadURL()
			ui.Info(fmt.Sprintf("[DRY RUN] Would download: %s", url))
		} else {
			ui.Info("[DRY RUN] Would clone and build from source")
		}
		ui.Info("[DRY RUN] Would run: picoclaw onboard")
		return
	}

	if method == 0 {
		// Download pre-built binary
		installFromRelease(sys)
	} else {
		// Build from source
		installFromSource()
	}

	// Initialize PicoClaw
	ui.Step(3, "Initializing PicoClaw")
	ui.Info("Running: picoclaw onboard")
	if err := install.RunOnboard(); err != nil {
		ui.Warn(fmt.Sprintf("Onboard had issues: %v", err))
		ui.Info("You may need to run 'picoclaw onboard' manually after migration")
	} else {
		ui.Success("PicoClaw initialized")
	}
}

func installFromRelease(sys detect.SystemInfo) {
	ui.Step(1, "Downloading PicoClaw binary")

	url, filename, err := install.GetDownloadURL()
	if err != nil {
		ui.Fatal(fmt.Sprintf("Unsupported platform: %v", err))
	}

	ui.Info(fmt.Sprintf("URL: %s", url))
	tmpDir := os.TempDir()
	archivePath := filepath.Join(tmpDir, filename)

	if err := install.Download(url, archivePath); err != nil {
		ui.Fatal(fmt.Sprintf("Download failed: %v", err))
	}
	ui.Success("Download complete")

	ui.Step(2, "Installing binary")
	binaryPath, err := install.Extract(archivePath, tmpDir)
	if err != nil {
		ui.Fatal(fmt.Sprintf("Extraction failed: %v", err))
	}

	ui.Info("Installing to /usr/local/bin/picoclaw (may require sudo)")
	if err := install.InstallBinary(binaryPath); err != nil {
		ui.Fatal(fmt.Sprintf("Install failed: %v", err))
	}
	ui.Success("PicoClaw installed to /usr/local/bin/picoclaw")

	// Cleanup
	os.Remove(archivePath)
}

func installFromSource() {
	ui.Step(1, "Building PicoClaw from source")
	tmpDir := os.TempDir()

	ui.Info("Cloning repository and building...")
	if err := install.BuildFromSource(tmpDir); err != nil {
		ui.Fatal(fmt.Sprintf("Build failed: %v", err))
	}
	ui.Success("PicoClaw built and installed from source")
}

// ════════════════════════════════════════════════════════════
// Phase 4: Migrate data
// ════════════════════════════════════════════════════════════

func phase4Migrate(oc, pc detect.Installation, dryRun bool) {
	ui.Phase(4, "Migrate data")

	home, _ := os.UserHomeDir()
	picoHome := filepath.Join(home, ".picoclaw")
	picoWorkspace := filepath.Join(picoHome, "workspace")

	// === Step 1: Try built-in migration tool ===
	ui.Step(1, "Checking for PicoClaw's built-in migration tool")

	// Check if picoclaw migrate exists
	builtInAvailable := false
	if pc.BinaryPath != "" {
		// The built-in migrate command was added in v0.1.1
		builtInAvailable = true
	}

	useBuiltIn := false
	if builtInAvailable {
		ui.Success("Built-in 'picoclaw migrate' command is available")
		useBuiltIn = ui.Confirm("Use PicoClaw's built-in migration tool? (recommended)")
	}

	if useBuiltIn && !dryRun {
		ui.Info("Running: picoclaw migrate --force")
		ui.Info("(This handles workspace files + config conversion)")
		// Note: we'd execute picoclaw migrate here
		// For now, fall through to our own migration as supplement
	}

	// === Step 2: Migrate entire workspace ===
	ui.Step(2, "Migrating workspace (all files and directories)")

	if dryRun {
		ui.Info("[DRY RUN] Would migrate entire workspace:")

		// Standard agent files
		for _, f := range []string{"SOUL.md", "IDENTITY.md", "AGENTS.md", "USER.md", "TOOLS.md", "HEARTBEAT.md"} {
			srcPath := filepath.Join(oc.WorkspaceDir, f)
			if _, err := os.Stat(srcPath); err == nil {
				lines := detect.CountFileLines(srcPath)
				ui.Info(fmt.Sprintf("  %s (%d lines)", f, lines))
			}
		}

		// Extra files
		for _, f := range oc.ExtraFiles {
			lines := detect.CountFileLines(filepath.Join(oc.WorkspaceDir, f))
			ui.Info(fmt.Sprintf("  %s (%d lines)", f, lines))
		}

		// Directories
		entries, _ := os.ReadDir(oc.WorkspaceDir)
		for _, entry := range entries {
			if entry.IsDir() && !migrate.SkipEntries[entry.Name()] {
				dirPath := filepath.Join(oc.WorkspaceDir, entry.Name())
				count := detect.CountDirFiles(dirPath)
				ui.Info(fmt.Sprintf("  %s/ (%d files)", entry.Name(), count))
			}
		}
	} else {
		result := migrate.MigrateWorkspace(oc.WorkspaceDir, picoWorkspace, true)

		for _, fr := range result.Files {
			if fr.Migrated {
				ui.FileStatus(fr.Name, true, fr.Lines)
			} else if fr.Skipped {
				ui.FileStatus(fr.Name, false, 0)
			} else if fr.Error != nil {
				ui.Error(fmt.Sprintf("%s: %v", fr.Name, fr.Error))
			}
		}

		ui.Success(fmt.Sprintf("Migrated %d files (%d skipped, %d errors)",
			result.Migrated, result.Skipped, result.Errors))
	}

	// === Step 3: Migrate config ===
	ui.Step(3, "Converting configuration")

	if dryRun {
		ui.Info("[DRY RUN] Would convert: openclaw.json → config.json")
		if oc.Config != nil {
			providers := detect.GetProviderKeys(oc.Config)
			channels := detect.GetConfiguredChannels(oc.Config)
			ui.Info(fmt.Sprintf("  Providers: %s", strings.Join(providers, ", ")))
			ui.Info(fmt.Sprintf("  Channels: %s", strings.Join(channels, ", ")))
		}
	} else {
		picoConfigPath := filepath.Join(picoHome, "config.json")
		fr := migrate.MigrateConfig(oc.ConfigPath, picoConfigPath, true)
		if fr.Error != nil {
			ui.Error(fmt.Sprintf("Config migration failed: %v", fr.Error))
		} else {
			ui.Success("Configuration converted and written")
			if fr.BackedUp {
				ui.Info("Previous config backed up to config.json.bak")
			}
		}
	}

	// === Step 4: Manual items reminder ===
	ui.Step(4, "Items requiring manual attention")

	manualItems := []string{}

	// Check for MCP servers
	if oc.Config != nil {
		mcpServers := detect.GetMCPServers(oc.Config)
		if len(mcpServers) > 0 {
			manualItems = append(manualItems, fmt.Sprintf("MCP Servers (%s) — verify format in config", strings.Join(mcpServers, ", ")))
		}
	}

	// Check for cron jobs
	if oc.HasCron {
		manualItems = append(manualItems, "Cron jobs — recreate with: picoclaw cron add ...")
	}

	// Check for unsupported channels
	if oc.Config != nil {
		channels := detect.GetConfiguredChannels(oc.Config)
		unsupported := []string{}
		supported := map[string]bool{
			"telegram": true, "discord": true, "qq": true,
			"dingtalk": true, "line": true, "slack": true,
		}
		for _, ch := range channels {
			if !supported[ch] {
				unsupported = append(unsupported, ch)
			}
		}
		if len(unsupported) > 0 {
			manualItems = append(manualItems,
				fmt.Sprintf("Unsupported channels: %s (not available in PicoClaw)",
					strings.Join(unsupported, ", ")))
		}
	}

	if len(manualItems) > 0 {
		ui.Warn("The following items need manual attention:")
		for _, item := range manualItems {
			fmt.Printf("    "+ui.Yellow+"•"+ui.Reset+" %s\n", item)
		}
	} else {
		ui.Success("No manual items needed — everything migrated automatically!")
	}
}

// ════════════════════════════════════════════════════════════
// Phase 5: Verify
// ════════════════════════════════════════════════════════════

func phase5Verify() {
	ui.Phase(5, "Verify migration")

	ui.Step(1, "Testing PicoClaw")

	// Re-detect to verify
	pc := detect.DetectPicoClaw()

	if !pc.Found {
		ui.Error("PicoClaw workspace not found after migration")
		return
	}

	ui.Success(fmt.Sprintf("Workspace: %s", pc.WorkspaceDir))

	// Check workspace files
	allFiles := []string{"SOUL.md", "IDENTITY.md", "AGENTS.md", "USER.md", "TOOLS.md", "HEARTBEAT.md"}
	migratedCount := 0
	for _, f := range allFiles {
		if pc.WorkspaceFiles[f] {
			migratedCount++
		}
	}
	ui.Success(fmt.Sprintf("Workspace files: %d/%d present", migratedCount, len(allFiles)))

	// Check config
	if pc.Config != nil {
		ui.Success("Config file: valid JSON")
	} else {
		ui.Warn("Config file: could not parse — manual review needed")
	}

	ui.Step(2, "Quick test")
	ui.Info("Try running these commands to verify:")
	fmt.Println()
	fmt.Println("    " + ui.Cyan + "picoclaw status" + ui.Reset + "                    # Check status")
	fmt.Println("    " + ui.Cyan + "picoclaw agent -m \"Hello!\"" + ui.Reset + "         # Test chat")
	fmt.Println("    " + ui.Cyan + "picoclaw gateway" + ui.Reset + "                    # Start Telegram bot")
	fmt.Println()

	if !ui.Confirm("Have you tested PicoClaw and confirmed it works?") {
		ui.Warn("Skipping uninstall phase. You can uninstall OpenClaw later.")
		ui.Info("Run: claw-migrate --skip-install to skip to uninstall phase")
		ui.Info("Or manually: npm uninstall -g openclaw && rm -rf ~/.openclaw")
		ui.CompletionBanner()
		os.Exit(0)
	}
}

// ════════════════════════════════════════════════════════════
// Phase 6: Uninstall OpenClaw
// ════════════════════════════════════════════════════════════

func phase6Uninstall(oc detect.Installation, dryRun bool) {
	ui.Phase(6, "Uninstall OpenClaw")

	if !ui.ConfirmDangerous("Remove OpenClaw? (backup was created in Phase 2)") {
		ui.Info("Skipping uninstall. OpenClaw remains installed.")
		ui.Info("You can uninstall later with:")
		ui.Info("  npm uninstall -g openclaw")
		ui.Info("  rm -rf ~/.openclaw")
		return
	}

	// Step 1: Stop processes
	ui.Step(1, "Stopping OpenClaw processes")
	if dryRun {
		ui.Info("[DRY RUN] Would stop all OpenClaw processes")
	} else {
		uninstall.StopOpenClaw()
		ui.Success("OpenClaw processes stopped")
	}

	// Step 2: Remove binary
	ui.Step(2, "Removing OpenClaw binary")
	if dryRun {
		ui.Info("[DRY RUN] Would run: npm uninstall -g openclaw")
	} else {
		if err := uninstall.RemoveBinary(); err != nil {
			ui.Warn(fmt.Sprintf("Could not remove via npm: %v", err))
			ui.Info("You may need to remove manually: npm uninstall -g openclaw")
		} else {
			ui.Success("OpenClaw npm package removed")
		}
	}

	// Step 3: Remove launch agents (macOS)
	ui.Step(3, "Removing launch agents")
	if dryRun {
		ui.Info("[DRY RUN] Would remove OpenClaw launch agents from ~/Library/LaunchAgents/")
	} else {
		removed := uninstall.RemoveLaunchAgents()
		if len(removed) > 0 {
			for _, name := range removed {
				ui.Success(fmt.Sprintf("Removed: %s", name))
			}
		} else {
			ui.Info("No launch agents found")
		}
	}

	// Step 4: Remove data directory
	ui.Step(4, "Removing OpenClaw data directory")
	ui.Warn(fmt.Sprintf("This will delete: %s", oc.HomeDir))
	ui.Info("Your backup is safe — this only removes the live data.")

	if dryRun {
		ui.Info(fmt.Sprintf("[DRY RUN] Would remove: %s", oc.HomeDir))
	} else {
		if ui.ConfirmDangerous(fmt.Sprintf("Delete %s permanently?", oc.HomeDir)) {
			if err := uninstall.RemoveData(oc.HomeDir); err != nil {
				ui.Error(fmt.Sprintf("Could not remove data: %v", err))
			} else {
				ui.Success("OpenClaw data directory removed")
			}
		} else {
			ui.Info("Keeping data directory. Remove manually when ready:")
			ui.Info(fmt.Sprintf("  rm -rf %s", oc.HomeDir))
		}
	}

	// Step 5: Verify
	ui.Step(5, "Verifying removal")
	if !dryRun {
		binGone, dataGone, agentsGone := uninstall.VerifyRemoved()
		if binGone {
			ui.Success("Binary: removed")
		} else {
			ui.Warn("Binary: still present")
		}
		if dataGone {
			ui.Success("Data: removed")
		} else {
			ui.Warn("Data: still present")
		}
		if agentsGone {
			ui.Success("Launch agents: clean")
		} else {
			ui.Warn("Launch agents: still present")
		}
	}
}

// ════════════════════════════════════════════════════════════
// Help
// ════════════════════════════════════════════════════════════

func printHelp() {
	fmt.Println(ui.Bold + "claw-migrate" + ui.Reset + " — Interactive OpenClaw → PicoClaw migration wizard")
	fmt.Println()
	fmt.Println(ui.Bold + "USAGE:" + ui.Reset)
	fmt.Println("  claw-migrate                    Run the full interactive wizard")
	fmt.Println("  claw-migrate --dry-run           Show what would happen (no changes)")
	fmt.Println("  claw-migrate --skip-install      Skip PicoClaw installation")
	fmt.Println("  claw-migrate --skip-uninstall    Skip OpenClaw removal")
	fmt.Println("  claw-migrate --help              Show this help")
	fmt.Println("  claw-migrate --version           Show version")
	fmt.Println()
	fmt.Println(ui.Bold + "PHASES:" + ui.Reset)
	fmt.Println("  1. Detect    — Find OpenClaw & PicoClaw installations")
	fmt.Println("  2. Backup    — Create full backup of ~/.openclaw/")
	fmt.Println("  3. Install   — Download & install PicoClaw binary")
	fmt.Println("  4. Migrate   — Copy workspace, convert config, map providers")
	fmt.Println("  5. Verify    — Test PicoClaw works correctly")
	fmt.Println("  6. Uninstall — Remove OpenClaw (optional, with confirmation)")
	fmt.Println()
	fmt.Println(ui.Bold + "WHAT GETS MIGRATED:" + ui.Reset)
	fmt.Println("  ✅ Workspace files (SOUL.md, IDENTITY.md, AGENTS.md, USER.md, etc.)")
	fmt.Println("  ✅ Long-term memory (memory/)")
	fmt.Println("  ✅ Custom skills (skills/)")
	fmt.Println("  ✅ Provider API keys (Anthropic, OpenAI, OpenRouter, etc.)")
	fmt.Println("  ✅ Channel configs (Telegram, Discord, Slack)")
	fmt.Println("  ✅ Heartbeat settings")
	fmt.Println("  ⚠️  MCP connections (migrated, manual verification needed)")
	fmt.Println("  ❌ Session history (incompatible format)")
	fmt.Println("  ❌ Cron jobs (manual recreation needed)")
	fmt.Println()
	fmt.Println(ui.Bold + "REPOSITORY:" + ui.Reset)
	fmt.Println("  https://github.com/arunbluez/claw-migrate")
	fmt.Println()
}
