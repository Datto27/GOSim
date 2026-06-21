package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/config"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background HTTP API server started with 'gosim start'",
	Long: `Sends SIGTERM to the process recorded by 'gosim start' and waits for it to
exit gracefully (up to 12s, matching the server's 10s drain window), then
removes the pid file.

Has no effect on a server running in the foreground via 'gosim serve' — stop
that with Ctrl+C instead.`,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStop(cmd.Context())
	},
}

func runStop(ctx context.Context) error {
	pidPath, err := config.DefaultPIDPath()
	if err != nil {
		return err
	}

	pid, alive := readAlivePID(pidPath)
	if !alive {
		fmt.Println("gosim is not running (no active background process found).")
		return nil
	}

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop: %w", err)
	}
	fmt.Printf("Stopping gosim (pid %d)…\n", pid)

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			os.Remove(pidPath)
			fmt.Println("Stopped.")
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("stop: pid %d did not exit within 12s — check it manually (e.g. 'kill -9 %d')", pid, pid)
}
