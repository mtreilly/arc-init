// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type shellStatus struct {
	shell     string
	written   bool
	skipped   bool
	rcWritten bool
	rcSkipped bool
	rcRemoved bool
	reason    string
}

func newShellCmd() *cobra.Command {
	var bash, zsh, fish, powershell bool
	var force bool
	var writeRC bool
	var uninstallRC bool
	var all bool

	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Initialize shell completions",
		Long: `Set up shell completions for arc commands.

Installs completion scripts for bash, zsh, fish, and PowerShell.
By default, detects your current shell from the SHELL environment variable.

Idempotent: Running multiple times is safe. Existing files are not overwritten
unless --force is used. RC file blocks are added once and not duplicated.`,
		Example: `  arc-init shell
  arc-init shell --all
  arc-init shell --bash --zsh
  arc-init shell --write-rc
  arc-init shell --uninstall-rc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !bash && !zsh && !fish && !powershell {
				if all {
					bash, zsh, fish = true, true, true
				} else {
					sh := detectShell()
					switch sh {
					case "bash":
						bash = true
					case "zsh":
						zsh = true
					case "fish":
						fish = true
					case "powershell":
						powershell = true
					default:
						bash, zsh = true, true
					}
				}
			}

			var statuses []shellStatus
			root := cmd.Root()

			if bash {
				status := shellStatus{shell: "bash"}
				if err := writeShellCompletion(&status, root, "bash", force); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "bash completion: %v\n", err)
				}
				if writeRC && !uninstallRC {
					if err := ensureShellRC(&status, "bash", force); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "bash RC: %v\n", err)
					}
				}
				if uninstallRC {
					if err := removeRCBlock(bashRCPath()); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "remove bash RC: %v\n", err)
					} else {
						status.rcRemoved = true
					}
				}
				statuses = append(statuses, status)
			}

			if zsh {
				status := shellStatus{shell: "zsh"}
				if err := writeShellCompletion(&status, root, "zsh", force); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "zsh completion: %v\n", err)
				}
				if writeRC && !uninstallRC {
					if err := ensureShellRC(&status, "zsh", force); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "zsh RC: %v\n", err)
					}
				}
				if uninstallRC {
					if err := removeRCBlock(zshRCPath()); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "remove zsh RC: %v\n", err)
					} else {
						status.rcRemoved = true
					}
				}
				statuses = append(statuses, status)
			}

			if fish {
				status := shellStatus{shell: "fish"}
				if err := writeShellCompletion(&status, root, "fish", force); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "fish completion: %v\n", err)
				}
				statuses = append(statuses, status)
			}

			if powershell {
				status := shellStatus{shell: "powershell"}
				if err := writeShellCompletion(&status, root, "powershell", force); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "powershell completion: %v\n", err)
				}
				statuses = append(statuses, status)
			}

			reportShellStatus(cmd, statuses, uninstallRC)
			return nil
		},
	}

	cmd.Flags().BoolVar(&bash, "bash", false, "Install bash completion")
	cmd.Flags().BoolVar(&zsh, "zsh", false, "Install zsh completion")
	cmd.Flags().BoolVar(&fish, "fish", false, "Install fish completion")
	cmd.Flags().BoolVar(&powershell, "powershell", false, "Install PowerShell completion")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&writeRC, "write-rc", false, "Append idempotent RC lines to enable completions")
	cmd.Flags().BoolVar(&uninstallRC, "uninstall-rc", false, "Remove RC lines previously added by arc")
	cmd.Flags().BoolVar(&all, "all", false, "Install completions for all supported shells")

	return cmd
}

func writeShellCompletion(status *shellStatus, root *cobra.Command, shell string, force bool) error {
	var (
		path string
		err  error
	)

	switch shell {
	case "bash":
		path, err = writeBashCompletion(root, force)
	case "zsh":
		path, err = writeZshCompletion(root, force)
	case "fish":
		path, err = writeFishCompletion(root, force)
	case "powershell":
		path, err = writePSCompletion(root, force)
	default:
		return fmt.Errorf("unknown shell: %s", shell)
	}

	if err != nil {
		return err
	}

	if path == "" {
		status.skipped = true
		status.reason = "completion file already exists (use --force to overwrite)"
	} else {
		status.written = true
	}

	return nil
}

func ensureShellRC(status *shellStatus, shell string, force bool) error {
	var path string
	var block string

	if shell == "bash" {
		path = bashRCPath()
		block = rcStart + "\n" + `# Arc bash completions
if [ -f "$HOME/.config/bash/completions/arc.bash" ]; then
  . "$HOME/.config/bash/completions/arc.bash"
fi` + "\n" + rcEnd + "\n"
	} else if shell == "zsh" {
		path = zshRCPath()
		block = rcStart + "\n" + `# Arc zsh completions
fpath+=(~/.zsh/completions)
autoload -Uz compinit
compinit` + "\n" + rcEnd + "\n"
	}

	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)

	if data, err := os.ReadFile(path); err == nil {
		content := string(data)
		if strings.Contains(content, rcStart) && strings.Contains(content, rcEnd) {
			status.rcSkipped = true
			status.reason = "RC block already present (use --force to update)"
			return nil
		}
	}

	if err := upsertRCBlock(path, block, force); err != nil {
		return err
	}

	status.rcWritten = true
	return nil
}

func reportShellStatus(cmd *cobra.Command, statuses []shellStatus, uninstalled bool) {
	if len(statuses) == 0 {
		return
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "=== Shell Completions Status ===")
	fmt.Fprintln(cmd.OutOrStdout())

	for _, s := range statuses {
		fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", strings.ToUpper(s.shell))

		if uninstalled {
			fmt.Fprintln(cmd.OutOrStdout(), "  RC block: REMOVED")
		} else if s.written {
			fmt.Fprintln(cmd.OutOrStdout(), "  Completions: INSTALLED")
		} else if s.skipped {
			fmt.Fprintf(cmd.OutOrStdout(), "  Completions: SKIPPED (already exists, %s)\n", s.reason)
		}

		if s.rcWritten {
			fmt.Fprintln(cmd.OutOrStdout(), "  RC block: ADDED")
		} else if s.rcSkipped {
			fmt.Fprintf(cmd.OutOrStdout(), "  RC block: SKIPPED (%s)\n", s.reason)
		}

		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Next steps:")
	fmt.Fprintln(cmd.OutOrStdout(), "  - If completions not working, restart your shell")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Use --force to overwrite existing files")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Use --write-rc to update shell RC files")
}

func writeBashCompletion(root *cobra.Command, force bool) (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "bash", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "arc.bash")
	if !force {
		if _, err := os.Stat(path); err == nil {
			return "", nil
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := root.GenBashCompletion(f); err != nil {
		return "", err
	}
	return path, nil
}

func writeZshCompletion(root *cobra.Command, force bool) (string, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".zsh", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "_arc")
	if !force {
		if _, err := os.Stat(path); err == nil {
			return "", nil
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := root.GenZshCompletion(f); err != nil {
		return "", err
	}
	return path, nil
}

func writeFishCompletion(root *cobra.Command, force bool) (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "fish", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "arc.fish")
	if !force {
		if _, err := os.Stat(path); err == nil {
			return "", nil
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := root.GenFishCompletion(f, true); err != nil {
		return "", err
	}
	return path, nil
}

func writePSCompletion(root *cobra.Command, force bool) (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "powershell")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "arc.ps1")
	if !force {
		if _, err := os.Stat(path); err == nil {
			return "", nil
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := root.GenPowerShellCompletionWithDesc(f); err != nil {
		return "", err
	}
	return path, nil
}

const rcStart = "# >>> arc init >>>"
const rcEnd = "# <<< arc init <<<"

func detectShell() string {
	sh := os.Getenv("SHELL")
	if strings.Contains(sh, "zsh") {
		return "zsh"
	}
	if strings.Contains(sh, "bash") {
		return "bash"
	}
	if strings.Contains(sh, "fish") {
		return "fish"
	}
	if strings.Contains(strings.ToLower(sh), "powershell") {
		return "powershell"
	}
	return ""
}

func bashRCPath() string {
	home, _ := os.UserHomeDir()
	rc := filepath.Join(home, ".bashrc")
	if _, err := os.Stat(rc); errors.Is(err, os.ErrNotExist) {
		return filepath.Join(home, ".bash_profile")
	}
	return rc
}

func zshRCPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".zshrc")
}

func removeRCBlock(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	start := strings.Index(s, rcStart)
	end := strings.Index(s, rcEnd)
	if start == -1 || end == -1 || end < start {
		return nil
	}
	end += len(rcEnd)
	s2 := strings.TrimSpace(s[:start]+s[end:]) + "\n"
	return os.WriteFile(path, []byte(s2), 0o644)
}

func upsertRCBlock(path, block string, force bool) error {
	var cur string
	if _, err := os.Stat(path); err == nil {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		cur = string(b)
		if strings.Contains(cur, rcStart) && strings.Contains(cur, rcEnd) {
			return nil
		}
		if !force {
			_ = os.WriteFile(path+".arc.bak", b, 0o644)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n" + block)
	return err
}
