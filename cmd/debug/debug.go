package debug

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var debugCmd = &cobra.Command{
	Use:   "debug <platform>",
	Short: "Stream CodePush log output from a connected device or simulator",
	Long: `Stream real-time CodePush log output from a connected Android device or iOS simulator.

Requires adb (Android) or xcrun (iOS) to be available on PATH.

Platform must be "android" or "ios".`,
	GroupID: cmd.GroupDebug,
	Args:    cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out
		switch args[0] {
		case "android":
			return runDebugAndroid(c.Context(), out)
		case "ios":
			return runDebugIOS(c.Context(), out)
		default:
			return fmt.Errorf("unknown platform %q: must be android or ios", args[0])
		}
	},
}

func init() {
	cmd.RootCmd.AddGroup(&cobra.Group{ID: cmd.GroupDebug, Title: "Developer Tools:"})
	cmd.RootCmd.AddCommand(debugCmd)
}

func runDebugAndroid(ctx context.Context, out *output.Writer) error {
	if _, err := exec.LookPath("adb"); err != nil {
		return errors.New("adb not found on PATH: install Android SDK platform-tools and ensure adb is available")
	}

	out.Info("Streaming CodePush logs from Android device (Ctrl-C to stop)...")

	// -v raw strips ADB's built-in timestamp so the prefix added below is not duplicated.
	// Tag filter CodePush:V *:S selects only CodePush-tagged lines at the logcat layer.
	c := exec.CommandContext(ctx, "adb", "logcat", "-v", "raw", "CodePush:V", "*:S")

	stdout, err := c.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating adb stdout pipe: %w", err)
	}

	if err := c.Start(); err != nil {
		return fmt.Errorf("starting adb logcat: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			ts := time.Now().Format("15:04:05.000")
			_, _ = fmt.Fprintf(os.Stdout, "[%s] %s\n", ts, scanner.Text())
		}
		done <- c.Wait()
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-done:
		return err
	}
}

func runDebugIOS(ctx context.Context, out *output.Writer) error {
	if _, err := exec.LookPath("xcrun"); err != nil {
		return errors.New("xcrun not found on PATH: install Xcode command-line tools")
	}

	out.Info("Streaming CodePush logs from iOS simulator (Ctrl-C to stop)...")

	c := exec.CommandContext(ctx, "xcrun", "simctl", "spawn", "booted", "log", "stream",
		"--predicate", `subsystem contains "codepush" OR process contains "CodePush"`)

	stdout, err := c.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating xcrun stdout pipe: %w", err)
	}

	if err := c.Start(); err != nil {
		return fmt.Errorf("starting xcrun log stream: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Unified log stream lines already include native timestamps; pass them through as-is.
			_, _ = fmt.Fprintln(os.Stdout, scanner.Text())
		}
		done <- c.Wait()
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-done:
		return err
	}
}
