package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/config"
)

var startPort int

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the HTTP API server in the background",
	Long: `Launches 'gosim serve' as a detached background process, redirecting its
output to a log file and recording its pid. Returns immediately.

Use 'gosim stop' to shut it down, or 'gosim serve' to run in the foreground
instead (e.g. to watch logs directly or under a process supervisor).`,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStart(cmd.Context())
	},
}

func init() {
	startCmd.Flags().IntVar(&startPort, "port", 0, "Port to listen on (default: value from config)")
}

func runStart(ctx context.Context) error {
	path := configPath
	if path == "" {
		p, err := config.DefaultConfigPath()
		if err != nil {
			return err
		}
		path = p
	}
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("%w\n\nRun 'gosim setup' to create a config file.", err)
	}

	pidPath, err := config.DefaultPIDPath()
	if err != nil {
		return err
	}
	if pid, alive := readAlivePID(pidPath); alive {
		return fmt.Errorf("start: gosim is already running (pid %d) — run 'gosim stop' first", pid)
	}

	port := startPort
	if port == 0 {
		port = cfg.APIPort
	}
	if port == 0 {
		port = 7700
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	serveArgs := []string{"serve", "--port", strconv.Itoa(port)}
	if configPath != "" {
		serveArgs = append(serveArgs, "--config", configPath)
	}

	logPath, err := config.DefaultLogPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("start: open log file: %w", err)
	}
	defer logFile.Close()

	child := exec.Command(exe, serveArgs...)
	child.Stdout = logFile
	child.Stderr = logFile
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := child.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
		return fmt.Errorf("start: write pid file: %w", err)
	}

	// Give it a moment, then confirm it didn't die immediately (bad config,
	// port already in use, DB unreachable, etc.) before reporting success.
	time.Sleep(500 * time.Millisecond)
	if !processAlive(child.Process.Pid) {
		os.Remove(pidPath)
		tail, _ := os.ReadFile(logPath)
		return fmt.Errorf("start: gosim exited immediately — see %s:\n%s", logPath, lastLines(string(tail), 10))
	}

	fmt.Printf("gosim started in background (pid %d), listening on localhost:%d\n", child.Process.Pid, port)
	fmt.Printf("  logs: %s\n", logPath)
	fmt.Println("  stop with: gosim stop")
	return nil
}

// processAlive reports whether a process with the given pid exists and is
// signalable. Sending signal 0 performs no action beyond the existence/
// permission check.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// readAlivePID reads the pid file at path and reports whether it names a
// still-running process, removing a stale file if not.
func readAlivePID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	if !processAlive(pid) {
		os.Remove(path)
		return 0, false
	}
	return pid, true
}

// lastLines returns at most n trailing non-empty lines of s.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
