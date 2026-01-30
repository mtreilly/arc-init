// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for arc-init.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arc-init",
		Short: "Initialize arc components",
		Long: `Initialize various arc components.

This command group provides setup wizards for different arc features:
  - system: Initialize global arc configuration (~/.config/arc/)
  - project: Initialize project-local configuration (.arc/config.yaml)
  - shell: Initialize shell completions (bash, zsh, fish, PowerShell)`,
		Example: `  arc init system --interactive
  arc init project --interactive
  arc init project --scaffold --gitignore
  arc init shell`,
	}

	cmd.AddCommand(
		newSystemCmd(),
		newProjectCmd(),
		newShellCmd(),
	)

	return cmd
}
