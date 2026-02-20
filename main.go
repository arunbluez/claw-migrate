package main

import (
	"encoding/json"
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

var version = "dev"

// Known outdated models and their recommended replacements
var modelUpgrades = map[string]string{
	"anthropic/claude-sonnet-4-5":              "anthropic/claude-sonnet-4-6",
	"anthropic/claude-3-5-sonnet":              "anthropic/claude-sonnet-4-6",
	"anthropic/claude-3-opus":                  "anthropic/claude-opus-4-6",
	"openai/gpt-4":                             "openai/gpt-5.2",
	"openai/gpt-4-turbo":                       "openai/gpt-5.2",
	"openai/gpt-4o":                            "openai/gpt-5.2",
	"openrouter/anthropic/claude-sonnet-4-5":   "openrouter/anthropic/claude-sonnet-4-6",
	"openrouter/anthropic/claude-3-5-sonnet":   "openrouter/anthropic/claude-sonnet-4-6",
}

func main() {
	dryRun := false
	skipInstall := false
	skipUninstall := false
	subcommand := ""

	args := []string{}
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
			fmt.Printf("claw-migrate %s\n", version)
			return
		default:
			if !strings.HasPrefix(arg, "-") {
				args = append(args, arg)
			}
		}
	}

	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "migrate":
		runMigrate(dryRun, skipInstall, skipUninstall)
	case "backup":
		runBackup()
	case "restore":
		runRestore()
	case "uninstall":
		runUninstallMenu()
	case "uninstall-openclaw":
		runUninstallOpenClaw()
	case "uninstall-picoclaw":
		runUninstallPicoClaw()
	case "":
		// Interactive menu
		ui.Banner()
		choice := ui.Choose("What would you like to do?", []string{
			"Migrate   — Full OpenClaw → PicoClaw migration",
			"Backup    — Create a backup of OpenClaw",
			"Restore   — Restore OpenClaw from a backup",
			"Uninstall — Remove OpenClaw or PicoClaw",
		})
		switch choice {
		case 0:
			runMigrate(dryRun, skipInstall, skipUninstall)
		case 1:
			runBackup()
		case 2:
			runRestore()
		case 3:
			runUninstallMenu()
		}
	default:
		ui.Error(fmt.Sprintf("Unknown command: %s", subcommand))
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Usage: claw-migrate [command] [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  migrate     Full OpenClaw → PicoClaw migration (default)")
	fmt.Println("  backup      Create a backup of ~/.openclaw/")
	fmt.Println("  restore     Restore OpenClaw from a backup")
	fmt.Println("  uninstall   Remove OpenClaw or PicoClaw")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --dry-run          Preview without making changes")
	fmt.Println("  --skip-install     Use existing PicoClaw installation")
	fmt.Println("  --skip-uninstall   Keep OpenClaw installed")
	fmt.Println("  --version          Show version")
	fmt.Println("  --help             Show this help")
	fmt.Println()
	fmt.Println("Run without arguments for interactive mode.")
}

// ════════════════════════════════════════════════════════════
// Standalone: Backup
// ════════════════════════════════════════════════════════════

func runBackup() {
	ui.Banner()
	ui.Phase(1, "Backup OpenClaw")

	oc := detect.DetectOpenClaw()
	if !oc.Found {
		ui.Error("OpenClaw installation not found at ~/.openclaw/")
		os.Exit(1)
	}

	ui.Found("Directory", oc.HomeDir)
	totalSize := detect.DirSize(oc.HomeDir)
	ui.Found("Size", detect.FormatSize(totalSize))
	doBackup(oc, false)

	ui.Success("Done!")
}

// ════════════════════════════════════════════════════════════
// Standalone: Restore
// ════════════════════════════════════════════════════════════

func runRestore() {
	ui.Banner()
	ui.Phase(1, "Restore OpenClaw from backup")

	backups := backup.ListBackups()
	if len(backups) == 0 {
		ui.Error("No backup files found (looking for ~/openclaw-backup-*.tar.gz)")
		os.Exit(1)
	}

	ui.Step(1, fmt.Sprintf("Found %d backup(s)", len(backups)))

	options := make([]string, len(backups))
	for i, b := range backups {
		options[i] = fmt.Sprintf("%s (%s)", b.Filename, backup.FormatSize(b.Size))
	}

	choice := ui.Choose("Which backup do you want to restore?", options)
	selected := backups[choice]

	ui.Warn(fmt.Sprintf("This will replace ~/.openclaw with the contents of %s", selected.Filename))
	if !ui.ConfirmDangerous("Proceed with restore?") {
		ui.Info("Restore cancelled.")
		return
	}

	// Verify
	ui.Step(2, "Verifying backup integrity")
	verifyErr := ui.SpinnerRun("Verifying backup...", func() error {
		return backup.VerifyBackup(selected.Path)
	})
	if verifyErr != nil {
		ui.Error(fmt.Sprintf("Backup is corrupted: %v", verifyErr))
		os.Exit(1)
	}
	ui.Success("Backup verified")

	// Restore
	ui.Step(3, "Restoring")
	restoreErr := ui.SpinnerRun("Restoring OpenClaw...", func() error {
		return backup.RestoreBackup(selected.Path)
	})
	if restoreErr != nil {
		ui.Error(fmt.Sprintf("Restore failed: %v", restoreErr))
		os.Exit(1)
	}

	ui.Success("OpenClaw restored from backup!")
	ui.Info("Run: openclaw status")
}

// ════════════════════════════════════════════════════════════
// Standalone: Uninstall
// ════════════════════════════════════════════════════════════

func runUninstallMenu() {
	ui.Banner()

	choice := ui.Choose("What do you want to uninstall?", []string{
		"OpenClaw  — Remove OpenClaw (binary + data)",
		"PicoClaw  — Remove PicoClaw (binary + data) for a fresh start",
	})

	switch choice {
	case 0:
		runUninstallOpenClaw()
	case 1:
		runUninstallPicoClaw()
	}
}

func runUninstallOpenClaw() {
	oc := detect.DetectOpenClaw()
	if !oc.Found && oc.BinaryPath == "" {
		ui.Error("OpenClaw installation not found")
		os.Exit(1)
	}

	// Offer backup first
	if oc.Found {
		ui.Warn("It's recommended to create a backup before uninstalling.")
		if ui.Confirm("Create a backup first?") {
			doBackup(oc, false)
		}
	}

	phase6Uninstall(oc, false)
	ui.Success("Done!")
}

func runUninstallPicoClaw() {
	home, _ := os.UserHomeDir()
	picoHome := filepath.Join(home, ".picoclaw")

	pc := detect.DetectPicoClaw()
	if !pc.Found && pc.BinaryPath == "" {
		ui.Error("PicoClaw installation not found")
		os.Exit(1)
	}

	ui.Phase(1, "Uninstall PicoClaw")

	if pc.BinaryPath != "" {
		ui.Found("Binary", pc.BinaryPath)
	}
	if pc.Found {
		ui.Found("Data", picoHome)
		totalSize := detect.DirSize(picoHome)
		ui.Found("Size", detect.FormatSize(totalSize))
	}

	ui.Warn("This will remove PicoClaw completely so you can start fresh.")
	if !ui.ConfirmDangerous("Uninstall PicoClaw?") {
		ui.Info("Cancelled.")
		return
	}

	// Stop processes
	ui.Step(1, "Stopping PicoClaw processes")
	uninstall.StopPicoClaw()
	ui.Success("Processes stopped")

	// Remove binary
	if pc.BinaryPath != "" {
		ui.Step(2, "Removing binary")
		if err := uninstall.RemovePicoClawBinary(); err != nil {
			ui.Warn(fmt.Sprintf("Could not remove binary: %v", err))
			ui.Info("You may need to manually delete: " + pc.BinaryPath)
		} else {
			ui.Success("Binary removed")
		}
	}

	// Remove launch agents (macOS)
	ui.Step(3, "Removing launch agents")
	if removed := uninstall.RemovePicoClawLaunchAgents(); len(removed) > 0 {
		ui.Success(fmt.Sprintf("Removed %d launch agent(s)", len(removed)))
	} else {
		ui.Info("No launch agents found")
	}

	// Remove data
	if pc.Found {
		ui.Step(4, "Removing data directory")
		ui.Warn(fmt.Sprintf("About to delete: %s", picoHome))

		if !ui.ConfirmDangerous("Delete all PicoClaw data?") {
			ui.Info("Data directory preserved at " + picoHome)
		} else {
			if err := uninstall.RemoveData(picoHome); err != nil {
				ui.Error(fmt.Sprintf("Could not remove data: %v", err))
			} else {
				ui.Success("PicoClaw data removed")
			}
		}
	}

	// Verify
	ui.Step(5, "Verifying removal")
	binaryGone, dataGone, _ := uninstall.VerifyPicoClawRemoved()
	if binaryGone && dataGone {
		ui.Success("PicoClaw completely removed")
	} else {
		if !binaryGone {
			ui.Warn("Binary still found — try: sudo rm " + pc.BinaryPath)
		}
		if !dataGone {
			ui.Warn("Data still found — try: rm -rf " + picoHome)
		}
	}

	fmt.Println()
	ui.Info("You can now run a fresh migration with: ./claw-migrate migrate")
}

// ════════════════════════════════════════════════════════════
// Full migration flow
// ════════════════════════════════════════════════════════════

func runMigrate(dryRun, skipInstall, skipUninstall bool) {
	ui.Banner()

	if dryRun {
		ui.Warn("DRY RUN mode — no changes will be made")
	}

	// Phase 1: Detect
	phase1Detect()
	oc := detect.DetectOpenClaw()
	pc := detect.DetectPicoClaw()
	sys := detect.GetSystemInfo()

	if !oc.Found {
		ui.Error("OpenClaw installation not found at ~/.openclaw/")
		ui.Info("Make sure OpenClaw is installed and has been initialized.")
		os.Exit(1)
	}

	showDetectionResults(oc, pc, sys)

	if !ui.Confirm("Ready to begin migration?") {
		ui.Info("Migration cancelled. No changes made.")
		return
	}

	// Phase 2: Backup
	phase2Backup(oc, dryRun)

	// Phase 3: Install PicoClaw
	if !skipInstall {
		phase3Install(pc, sys, dryRun)
	} else {
		ui.Phase(3, "Install PicoClaw (skipped)")
		ui.Info("--skip-install flag set")
	}

	pc = detect.DetectPicoClaw()

	// Phase 4: Migrate
	phase4Migrate(oc, pc, dryRun)

	// Phase 5: Verify
	phase5Verify()

	// Phase 6: Uninstall
	if !skipUninstall {
		phase6Uninstall(oc, dryRun)
	} else {
		ui.Phase(6, "Uninstall OpenClaw (skipped)")
		ui.Info("--skip-uninstall flag set. You can uninstall later with:")
		ui.Info("  npm uninstall -g openclaw && rm -rf ~/.openclaw")
	}

	ui.CompletionBanner()
}

// ════════════════════════════════════════════════════════════
// Phase 1: Detect
// ════════════════════════════════════════════════════════════

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

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

	// Config summary
	ui.Step(3, "Configuration")
	if oc.Config != nil {
		ui.Found("Config file", fmt.Sprintf("%s (%s)", oc.ConfigPath, detect.FormatSize(oc.ConfigSummary.ConfigFileSize)))

		if oc.ConfigSummary.DefaultModel != "" {
			// Check if model is outdated
			if upgrade, found := modelUpgrades[oc.ConfigSummary.DefaultModel]; found {
				ui.Warn(fmt.Sprintf("Default model          %s (outdated → %s available)", oc.ConfigSummary.DefaultModel, upgrade))
			} else {
				ui.Found("Default model", oc.ConfigSummary.DefaultModel)
			}
		}
		if oc.ConfigSummary.MaxTokens > 0 {
			ui.Found("Max tokens", fmt.Sprintf("%d", oc.ConfigSummary.MaxTokens))
		}

		providers := detect.GetProviderKeys(oc.Config)
		if len(providers) > 0 {
			ui.Found("Providers", strings.Join(providers, ", "))
		}

		channels := detect.GetConfiguredChannels(oc.Config)
		if len(channels) > 0 {
			ui.Found("Channels", strings.Join(channels, ", "))
		}

		mcpServers := detect.GetMCPServers(oc.Config)
		if len(mcpServers) > 0 {
			ui.Found("MCP Servers", strings.Join(mcpServers, ", "))
		}

		if oc.ConfigSummary.HeartbeatEnabled {
			ui.Found("Heartbeat", fmt.Sprintf("enabled (every %d min)", oc.ConfigSummary.HeartbeatInterval))
		}
	} else {
		ui.NotFound("Config file")
	}

	// Workspace — standard agent files
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

	// Extra files
	if len(oc.ExtraFiles) > 0 {
		ui.Step(5, fmt.Sprintf("Workspace — custom files (%d)", len(oc.ExtraFiles)))
		for _, f := range oc.ExtraFiles {
			lines := detect.CountFileLines(filepath.Join(oc.WorkspaceDir, f))
			ui.FileStatus(f, true, lines)
		}
	}

	// Standard directories
	ui.Step(6, "Workspace — standard directories")
	stdDirs := []struct {
		name string
		has  bool
	}{
		{"memory", oc.HasMemory},
		{"skills", oc.HasSkills},
		{"scripts", dirExists(filepath.Join(oc.WorkspaceDir, "scripts"))},
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

	// Project directories
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

	// PicoClaw status
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
	doBackup(oc, dryRun)
}

func doBackup(oc detect.Installation, dryRun bool) {
	ui.Step(1, "Creating full backup of ~/.openclaw/")

	if dryRun {
		ui.Info("[DRY RUN] Would create backup: ~/openclaw-backup-YYYYMMDD-HHMMSS.tar.gz")
		return
	}

	var result backup.Result
	err := ui.SpinnerRun("Creating backup (this may take a minute)...", func() error {
		result = backup.CreateBackup(oc.HomeDir)
		if !result.Success {
			return result.Error
		}
		return nil
	})

	if err != nil {
		ui.Error(fmt.Sprintf("Backup failed: %v", err))
		if !ui.ConfirmDangerous("Continue WITHOUT backup? (not recommended)") {
			ui.Info("Migration cancelled.")
			os.Exit(1)
		}
		return
	}

	ui.Success(fmt.Sprintf("Backup created: %s (%s)", result.Path, backup.FormatSize(result.Size)))

	// Verify
	ui.Step(2, "Verifying backup integrity")
	verifyErr := ui.SpinnerRun("Verifying...", func() error {
		return backup.VerifyBackup(result.Path)
	})
	if verifyErr != nil {
		ui.Warn(fmt.Sprintf("Backup verification warning: %v", verifyErr))
	} else {
		ui.Success("Backup verified successfully")
	}
}

// ════════════════════════════════════════════════════════════
// Phase 3: Install PicoClaw
// ════════════════════════════════════════════════════════════

func phase3Install(pc detect.Installation, sys detect.SystemInfo, dryRun bool) {
	ui.Phase(3, "Install PicoClaw")

	// Fetch latest version
	ui.Step(1, "Checking latest PicoClaw release")
	var fetchedVersion string
	ui.SpinnerRun("Fetching latest version...", func() error {
		fetchedVersion = install.FetchLatestVersion()
		return nil
	})
	ui.Found("Latest version", "v"+fetchedVersion)

	// Already installed?
	if pc.BinaryPath != "" {
		ui.Success(fmt.Sprintf("PicoClaw already installed: %s", pc.BinaryPath))
		if pc.Version != "" {
			ui.Info(fmt.Sprintf("Version: %s", pc.Version))
		}
		if ui.Confirm("Skip installation and use existing PicoClaw?") {
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

	method := ui.Choose("How would you like to install PicoClaw?", []string{
		fmt.Sprintf("Download pre-built binary (%s, recommended)", install.VersionTag()),
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
		installFromRelease(sys)
	} else {
		installFromSource()
	}

	// Initialize
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

	dlErr := ui.SpinnerRun("Downloading...", func() error {
		return install.Download(url, archivePath)
	})
	if dlErr != nil {
		ui.Fatal(fmt.Sprintf("Download failed: %v", dlErr))
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
	ui.Success("PicoClaw installed")

	os.Remove(archivePath)
}

func installFromSource() {
	ui.Step(1, "Building PicoClaw from source")
	tmpDir := os.TempDir()

	err := ui.SpinnerRun("Cloning and building (this may take a few minutes)...", func() error {
		return install.BuildFromSource(tmpDir)
	})
	if err != nil {
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

	// Step 1: Check built-in migration tool
	ui.Step(1, "Checking for PicoClaw's built-in migration tool")

	builtInAvailable := pc.BinaryPath != ""
	useBuiltIn := false
	if builtInAvailable {
		ui.Success("Built-in 'picoclaw migrate' command is available")
		useBuiltIn = ui.Confirm("Use PicoClaw's built-in migration tool? (recommended)")
	}

	if useBuiltIn && !dryRun {
		ui.Info("Running: picoclaw migrate --force")
	}

	// Step 2: Migrate workspace — condensed output
	ui.Step(2, "Migrating workspace (all files and directories)")

	if dryRun {
		fileCount := 0
		dirCount := 0
		entries, _ := os.ReadDir(oc.WorkspaceDir)
		for _, entry := range entries {
			if migrate.SkipEntries[entry.Name()] {
				continue
			}
			if entry.IsDir() {
				dirCount++
				dirPath := filepath.Join(oc.WorkspaceDir, entry.Name())
				fileCount += detect.CountDirFiles(dirPath)
			} else {
				fileCount++
			}
		}
		ui.Info(fmt.Sprintf("[DRY RUN] Would migrate %d files across %d directories", fileCount, dirCount))
	} else {
		var result migrate.Result
		ui.SpinnerRun("Copying workspace files...", func() error {
			result = migrate.MigrateWorkspace(oc.WorkspaceDir, picoWorkspace, true)
			return nil
		})

		ui.Success(fmt.Sprintf("Migrated %d files (%d skipped, %d errors)",
			result.Migrated, result.Skipped, result.Errors))

		// Only show individual files if there were errors
		if result.Errors > 0 {
			for _, fr := range result.Files {
				if fr.Error != nil {
					ui.Error(fmt.Sprintf("  %s: %v", fr.Name, fr.Error))
				}
			}
		}
	}

	// Step 3: Migrate config
	ui.Step(3, "Converting configuration")

	if dryRun {
		ui.Info("[DRY RUN] Would convert: openclaw.json → config.json")
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

	// Step 4: Model version check
	ui.Step(4, "Checking model version")
	checkModelVersion(oc, picoHome, dryRun)

	// Step 5: Manual items
	ui.Step(5, "Items requiring manual attention")

	manualItems := []string{}

	if oc.Config != nil {
		mcpServers := detect.GetMCPServers(oc.Config)
		if len(mcpServers) > 0 {
			manualItems = append(manualItems, fmt.Sprintf("MCP Servers (%s) — verify format in config", strings.Join(mcpServers, ", ")))
		}
	}

	if oc.HasCron {
		manualItems = append(manualItems, "Cron jobs — recreate with: picoclaw cron add ...")
	}

	if oc.Config != nil {
		channels := detect.GetConfiguredChannels(oc.Config)
		unsupported := []string{}
		supported := map[string]bool{
			"telegram": true, "discord": true, "qq": true,
			"dingtalk": true, "line": true, "slack": true,
			"feishu": true, "onebot": true,
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
		ui.Success("No manual items — everything migrated automatically!")
	}
}

// checkModelVersion warns about outdated models and offers upgrade
func checkModelVersion(oc detect.Installation, picoHome string, dryRun bool) {
	currentModel := extractModelString(oc.Config)

	if currentModel == "" {
		ui.Info("No default model detected in config")
		return
	}

	if upgrade, found := modelUpgrades[currentModel]; found {
		ui.Warn(fmt.Sprintf("Current model: %s (outdated)", currentModel))
		ui.Info(fmt.Sprintf("Recommended:   %s", upgrade))

		if !dryRun {
			if ui.Confirm(fmt.Sprintf("Update model to %s?", upgrade)) {
				picoConfigPath := filepath.Join(picoHome, "config.json")
				if err := updateModelInConfig(picoConfigPath, upgrade); err != nil {
					ui.Error(fmt.Sprintf("Could not update model: %v", err))
				} else {
					ui.Success(fmt.Sprintf("Model updated to %s", upgrade))
				}
			} else {
				ui.Info(fmt.Sprintf("Keeping %s — you can change later in ~/.picoclaw/config.json", currentModel))
			}
		} else {
			ui.Info(fmt.Sprintf("[DRY RUN] Would offer to upgrade to %s", upgrade))
		}
	} else {
		ui.Success(fmt.Sprintf("Model: %s (current)", currentModel))
	}
}

// extractModelString gets the model name from OpenClaw config, handling both string and object formats
func extractModelString(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	// Try agent.model
	if agent, ok := config["agent"].(map[string]interface{}); ok {
		if model, ok := agent["model"]; ok {
			switch m := model.(type) {
			case string:
				return m
			case map[string]interface{}:
				for _, key := range []string{"primary", "name", "model", "default"} {
					if v, ok := m[key].(string); ok && v != "" {
						return v
					}
				}
			}
		}
	}

	// Try agents.defaults.model
	if agents, ok := config["agents"].(map[string]interface{}); ok {
		if defaults, ok := agents["defaults"].(map[string]interface{}); ok {
			if model, ok := defaults["model"]; ok {
				switch m := model.(type) {
				case string:
					return m
				case map[string]interface{}:
					for _, key := range []string{"primary", "name", "model", "default"} {
						if v, ok := m[key].(string); ok && v != "" {
							return v
						}
					}
				}
			}
		}
	}

	return ""
}

func updateModelInConfig(configPath, newModel string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		return err
	}

	if agents, ok := configMap["agents"].(map[string]interface{}); ok {
		if defaults, ok := agents["defaults"].(map[string]interface{}); ok {
			defaults["model"] = newModel
		}
	}

	out, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, out, 0644)
}

// ════════════════════════════════════════════════════════════
// Phase 5: Verify
// ════════════════════════════════════════════════════════════

func phase5Verify() {
	ui.Phase(5, "Verify migration")

	home, _ := os.UserHomeDir()
	picoWorkspace := filepath.Join(home, ".picoclaw", "workspace")
	picoConfig := filepath.Join(home, ".picoclaw", "config.json")

	ui.Step(1, "Checking PicoClaw workspace")

	// Check workspace exists
	if _, err := os.Stat(picoWorkspace); os.IsNotExist(err) {
		ui.Error("PicoClaw workspace not found!")
		return
	}
	ui.Success("Workspace directory exists")

	// Count files
	fileCount := detect.CountDirFiles(picoWorkspace)
	size := detect.DirSize(picoWorkspace)
	ui.Found("Workspace", fmt.Sprintf("%d files (%s)", fileCount, detect.FormatSize(size)))

	// Check config
	if _, err := os.Stat(picoConfig); err == nil {
		ui.Success("Configuration file exists")
	} else {
		ui.Warn("Configuration file missing")
	}

	// Check key workspace files
	ui.Step(2, "Checking key files")
	keyFiles := []string{"SOUL.md", "IDENTITY.md", "AGENTS.md"}
	allGood := true
	for _, f := range keyFiles {
		path := filepath.Join(picoWorkspace, f)
		if _, err := os.Stat(path); err == nil {
			lines := detect.CountFileLines(path)
			ui.FileStatus(f, true, lines)
		} else {
			ui.FileStatus(f, false, 0)
			allGood = false
		}
	}

	if allGood {
		ui.Success("All key files present")
	}

	// Suggested test commands
	ui.Step(3, "Test your PicoClaw installation")
	ui.Info("Try these commands:")
	fmt.Println()
	fmt.Println("    " + ui.Cyan + "picoclaw status" + ui.Reset + "          # Check status")
	fmt.Println("    " + ui.Cyan + "picoclaw agent" + ui.Reset + "           # Chat with your agent")
	fmt.Println("    " + ui.Cyan + "picoclaw gateway" + ui.Reset + "         # Start the gateway")
	fmt.Println()
}

// ════════════════════════════════════════════════════════════
// Phase 6: Uninstall OpenClaw
// ════════════════════════════════════════════════════════════

func phase6Uninstall(oc detect.Installation, dryRun bool) {
	ui.Phase(6, "Uninstall OpenClaw")

	ui.Warn("This will remove OpenClaw completely:")
	fmt.Printf("    "+ui.Yellow+"•"+ui.Reset+" Binary: %s\n", oc.BinaryPath)
	fmt.Printf("    "+ui.Yellow+"•"+ui.Reset+" Data: %s\n", oc.HomeDir)

	if !ui.ConfirmDangerous("Uninstall OpenClaw?") {
		ui.Info("OpenClaw preserved. You can uninstall later with:")
		ui.Info("  npm uninstall -g openclaw && rm -rf ~/.openclaw")
		return
	}

	if dryRun {
		ui.Info("[DRY RUN] Would uninstall OpenClaw")
		return
	}

	// Stop processes
	ui.Step(1, "Stopping OpenClaw processes")
	uninstall.StopOpenClaw()
	ui.Success("Processes stopped")

	// Remove binary
	ui.Step(2, "Removing binary")
	if err := uninstall.RemoveBinary(); err != nil {
		ui.Warn(fmt.Sprintf("Could not remove binary: %v", err))
	} else {
		ui.Success("Binary removed")
	}

	// Remove launch agents (macOS)
	ui.Step(3, "Removing launch agents")
	if removed := uninstall.RemoveLaunchAgents(); len(removed) > 0 {
		ui.Success(fmt.Sprintf("Removed %d launch agent(s)", len(removed)))
	} else {
		ui.Info("No launch agents found")
	}

	// Remove data
	ui.Step(4, "Removing data directory")
	ui.Warn(fmt.Sprintf("About to delete: %s", oc.HomeDir))

	if !ui.ConfirmDangerous("Delete all OpenClaw data? (backup was created in Phase 2)") {
		ui.Info("Data directory preserved.")
		return
	}

	if err := uninstall.RemoveData(oc.HomeDir); err != nil {
		ui.Error(fmt.Sprintf("Could not remove data: %v", err))
	} else {
		ui.Success("OpenClaw data removed")
	}

	// Verify
	ui.Step(5, "Verifying removal")
	binaryGone, dataGone, _ := uninstall.VerifyRemoved()
	if binaryGone && dataGone {
		ui.Success("OpenClaw completely removed")
	} else {
		ui.Warn("Some traces of OpenClaw may remain")
	}
}