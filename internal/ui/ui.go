package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Magenta   = "\033[35m"
	Cyan      = "\033[36m"
	White     = "\033[37m"
	BgBlue    = "\033[44m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgRed     = "\033[41m"
	BgMagenta = "\033[45m"
)

var reader = bufio.NewReader(os.Stdin)

// Banner prints the CLI banner
func Banner() {
	fmt.Println()
	fmt.Println(Cyan + Bold + "  ╔═══════════════════════════════════════════════════════════╗" + Reset)
	fmt.Println(Cyan + Bold + "  ║                                                           ║" + Reset)
	fmt.Println(Cyan + Bold + "  ║" + Reset + "   🦞 → 🦐  " + Bold + "claw-migrate" + Reset + "                                  " + Cyan + Bold + "║" + Reset)
	fmt.Println(Cyan + Bold + "  ║" + Reset + "   " + Dim + "OpenClaw → PicoClaw Migration Wizard" + Reset + "                   " + Cyan + Bold + "║" + Reset)
	fmt.Println(Cyan + Bold + "  ║                                                           ║" + Reset)
	fmt.Println(Cyan + Bold + "  ╚═══════════════════════════════════════════════════════════╝" + Reset)
	fmt.Println()
}

// Phase prints a phase header
func Phase(number int, title string) {
	fmt.Println()
	fmt.Printf(Bold+BgBlue+White+" PHASE %d "+Reset+Bold+" %s"+Reset+"\n", number, title)
	fmt.Println(Blue + "  " + strings.Repeat("─", 55) + Reset)
}

// Step prints a numbered step
func Step(number int, text string) {
	fmt.Printf("\n  "+Cyan+Bold+"[%d]"+Reset+" %s\n", number, text)
}

// Info prints an info message
func Info(msg string) {
	fmt.Println("  " + Dim + "ℹ  " + msg + Reset)
}

// Success prints a success message
func Success(msg string) {
	fmt.Println("  " + Green + "✅ " + msg + Reset)
}

// Warn prints a warning message
func Warn(msg string) {
	fmt.Println("  " + Yellow + "⚠️  " + msg + Reset)
}

// Error prints an error message
func Error(msg string) {
	fmt.Println("  " + Red + "❌ " + msg + Reset)
}

// Fatal prints error and exits
func Fatal(msg string) {
	Error(msg)
	os.Exit(1)
}

// Found prints a detection result
func Found(label, value string) {
	fmt.Printf("  "+Green+"✓"+Reset+" %-25s %s\n", label, Bold+value+Reset)
}

// NotFound prints a missing detection result
func NotFound(label string) {
	fmt.Printf("  "+Red+"✗"+Reset+" %-25s %s\n", label, Dim+"not found"+Reset)
}

// FileStatus prints file migration status
func FileStatus(name string, exists bool, lines int) {
	if exists {
		fmt.Printf("  "+Green+"  ✓"+Reset+" %-25s %s\n", name, Dim+fmt.Sprintf("(%d lines)", lines)+Reset)
	} else {
		fmt.Printf("  "+Yellow+"  ○"+Reset+" %-25s %s\n", name, Dim+"skipped (not found in source)"+Reset)
	}
}

// Confirm asks a yes/no question, returns true for yes
func Confirm(question string) bool {
	fmt.Printf("\n  "+Yellow+"?"+Reset+" %s "+Dim+"[Y/n]"+Reset+" ", question)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}

// ConfirmDangerous asks a yes/no question defaulting to no
func ConfirmDangerous(question string) bool {
	fmt.Printf("\n  "+Red+"⚠"+Reset+" %s "+Dim+"[y/N]"+Reset+" ", question)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// Prompt asks for text input
func Prompt(question string, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("\n  "+Yellow+"?"+Reset+" %s "+Dim+"[%s]"+Reset+" ", question, defaultVal)
	} else {
		fmt.Printf("\n  "+Yellow+"?"+Reset+" %s ", question)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// PromptSecret asks for secret input (shows dots)
func PromptSecret(question string) string {
	fmt.Printf("\n  "+Yellow+"🔑"+Reset+" %s: ", question)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// Choose presents numbered options and returns the selection index
func Choose(question string, options []string) int {
	fmt.Printf("\n  "+Yellow+"?"+Reset+" %s\n", question)
	for i, opt := range options {
		fmt.Printf("    "+Cyan+"%d)"+Reset+" %s\n", i+1, opt)
	}
	for {
		fmt.Printf("  "+Dim+"  Enter choice [1-%d]:"+Reset+" ", len(options))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(options) {
			return choice - 1
		}
		fmt.Println("  " + Red + "  Invalid choice, try again" + Reset)
	}
}

// Progress prints a progress bar
func Progress(current, total int, label string) {
	width := 30
	filled := (current * width) / total
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := (current * 100) / total
	fmt.Printf("\r  "+Cyan+"  [%s]"+Reset+" %3d%%  %s", bar, pct, label)
	if current == total {
		fmt.Println()
	}
}

// Spinner characters for animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerFrame returns the spinner character for a given tick
func SpinnerFrame(tick int) string {
	return Cyan + spinnerFrames[tick%len(spinnerFrames)] + Reset
}

// SpinnerRun runs a function with an animated spinner. Returns the function's error.
func SpinnerRun(label string, fn func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	tick := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			// Clear spinner line and show result
			fmt.Printf("\r  %-60s\r", "")
			return err
		case <-ticker.C:
			fmt.Printf("\r  %s %s", SpinnerFrame(tick), label)
			tick++
		}
	}
}

// Divider prints a thin divider
func Divider() {
	fmt.Println("  " + Dim + strings.Repeat("─", 55) + Reset)
}

// Summary prints a key-value summary line
func Summary(key, value string) {
	fmt.Printf("  %-28s %s\n", Dim+key+Reset, value)
}

// Box prints text in a box
func Box(title string, lines []string) {
	maxLen := len(title)
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	w := maxLen + 4
	fmt.Println()
	fmt.Println("  " + Dim + "┌" + strings.Repeat("─", w) + "┐" + Reset)
	fmt.Printf("  "+Dim+"│"+Reset+" "+Bold+"%-*s"+Reset+" "+Dim+"│"+Reset+"\n", w-2, title)
	fmt.Println("  " + Dim + "├" + strings.Repeat("─", w) + "┤" + Reset)
	for _, l := range lines {
		fmt.Printf("  "+Dim+"│"+Reset+" %-*s "+Dim+"│"+Reset+"\n", w-2, l)
	}
	fmt.Println("  " + Dim + "└" + strings.Repeat("─", w) + "┘" + Reset)
}

// CompletionBanner prints the final success banner
func CompletionBanner() {
	fmt.Println()
	fmt.Println(Green + Bold + "  ╔═══════════════════════════════════════════════════════════╗" + Reset)
	fmt.Println(Green + Bold + "  ║                                                           ║" + Reset)
	fmt.Println(Green + Bold + "  ║" + Reset + "   🦐  " + Bold + Green + "Migration Complete!" + Reset + "                                " + Green + Bold + "║" + Reset)
	fmt.Println(Green + Bold + "  ║                                                           ║" + Reset)
	fmt.Println(Green + Bold + "  ║" + Reset + "   Your PicoClaw assistant is ready to go.                 " + Green + Bold + "║" + Reset)
	fmt.Println(Green + Bold + "  ║" + Reset + "   Run: " + Cyan + "picoclaw gateway" + Reset + " to start!                       " + Green + Bold + "║" + Reset)
	fmt.Println(Green + Bold + "  ║                                                           ║" + Reset)
	fmt.Println(Green + Bold + "  ╚═══════════════════════════════════════════════════════════╝" + Reset)
	fmt.Println()
}