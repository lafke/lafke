package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lafke/lafke/internal/engine"
	"github.com/spf13/cobra"
)

func newEngineCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Install and inspect the source-code analysis engine",
		Long: `The analysis engine is a separate executable, released on its own schedule from
` + engine.ReleasesRepo + `. install downloads the newest engine that speaks this
lafke's protocol, checks its sha256 and its GitHub provenance attestation, and keeps it
in the user cache. lafke finds it there when no engine sits next to it or is set with
` + engine.EnvPath + `. update also removes an engine left next to the executable by older
releases, since that one would take precedence.

Set ` + engine.EnvVersion + ` to pin a version for a run. Commands that do not analyze source
code work without the engine.`,
	}
	cmd.AddCommand(newEngineInstallCommand(opts, "install", "Download and install the engine"))
	cmd.AddCommand(newEngineInstallCommand(opts, "update", "Install the newest compatible engine and remove older ones"))
	cmd.AddCommand(newEngineStatusCommand())
	cmd.AddCommand(newEngineListCommand(opts))
	return cmd
}

func newEngineInstallCommand(opts *options, use, short string) *cobra.Command {
	var version string
	var skipAttestation bool
	update := use == "update"
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.offline || opts.network == "offline" {
				return opErr(errors.New("cannot download the engine in offline mode"))
			}
			out := cmd.OutOrStdout()
			in, err := engine.Install(cmd.Context(), engine.InstallOptions{
				Version:         version,
				SkipAttestation: skipAttestation,
				Token:           githubToken(),
				Progress:        func(stage string) { fmt.Fprintln(cmd.ErrOrStderr(), stage) },
			})
			if err != nil {
				return opErr(err)
			}
			if skipAttestation {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning: attestation verification skipped")
			}
			fmt.Fprintf(out, "installed lafke-engine %s (protocol %d)\n  %s\n", in.Version, in.Protocol, in.Path)
			if update {
				removed, err := engine.Remove(in.Version)
				for _, v := range removed {
					fmt.Fprintf(out, "removed lafke-engine %s\n", v)
				}
				if err != nil {
					return opErr(err)
				}
				if bundled, ok := engine.Bundled(); ok {
					// It would win over the engine just installed, so it has to go.
					if err := os.Remove(bundled); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s still takes precedence over the installed engine; remove it manually: %v\n", bundled, err)
					} else {
						fmt.Fprintf(out, "removed %s\n", bundled)
					}
				}
			} else {
				warnBundled(cmd)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&skipAttestation, "skip-attestation", false, "install without verifying the GitHub attestation (the sha256 is still checked)")
	if !update {
		cmd.Flags().StringVar(&version, "version", "", "engine release tag to install (default: newest compatible)")
	}
	return cmd
}

func newEngineStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show which engine lafke uses and which are installed",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "protocol: %d\n", engine.ProtocolVersion)
			path, err := engine.Locate()
			if err != nil {
				fmt.Fprintf(out, "engine:   not found (%v)\n", err)
			} else {
				version, cerr := engine.CheckVersion()
				if cerr != nil {
					fmt.Fprintf(out, "engine:   %s (%v)\n", path, cerr)
				} else {
					fmt.Fprintf(out, "engine:   %s (%s)\n", path, version)
				}
			}
			warnBundled(cmd)
			installed := engine.List()
			fmt.Fprintf(out, "installed in %s:\n", engine.InstallDir())
			if len(installed) == 0 {
				fmt.Fprintln(out, "  none")
			}
			for _, in := range installed {
				fmt.Fprintf(out, "  %s\n", in.Version)
			}
			return nil
		},
	}
}

func newEngineListCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the published engine releases that speak this lafke's protocol",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.offline || opts.network == "offline" {
				return opErr(errors.New("cannot list engine releases in offline mode"))
			}
			releases, err := engine.Remote(cmd.Context(), githubToken())
			if err != nil {
				return opErr(err)
			}
			out := cmd.OutOrStdout()
			if len(releases) == 0 {
				fmt.Fprintf(out, "no engine release speaks protocol %d\n", engine.ProtocolVersion)
				return nil
			}
			installed := map[string]bool{}
			for _, in := range engine.List() {
				installed[in.Version] = true
			}
			for i, m := range releases {
				var notes []string
				if i == 0 {
					notes = append(notes, "newest")
				}
				if installed[m.Version] {
					notes = append(notes, "installed")
				}
				line := m.Version
				if len(notes) > 0 {
					line += " (" + strings.Join(notes, ", ") + ")"
				}
				fmt.Fprintln(out, line)
			}
			return nil
		},
	}
}

// warnBundled tells the user when an engine next to the executable overrides installed ones.
func warnBundled(cmd *cobra.Command) {
	if bundled, ok := engine.Bundled(); ok {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s takes precedence over installed engines; run `lafke engine update` to replace it\n", bundled)
	}
}

// githubToken returns the token used for GitHub requests, if the environment has one.
func githubToken() string {
	if t := os.Getenv("GH_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("GITHUB_TOKEN")
}
