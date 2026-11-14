package cli

import (
	"fmt"
	"strings"

	"github.com/lafke/cli/internal/diff"
	"github.com/lafke/cli/internal/finding"
	"github.com/lafke/cli/internal/lock"
	"github.com/lafke/cli/internal/policy"
	"github.com/lafke/cli/internal/scanner"
	"github.com/spf13/cobra"
)

func newDiffCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "diff [path]",
		Short: "Show trust drift against lafke.lock",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := projectPath(args)
			pol, err := policy.LoadWithProfile(opts.policy, opts.profile)
			if err != nil {
				return opErr(err)
			}
			net, err := scanOptions(opts, pol)
			if err != nil {
				return err
			}

			lockPath := lock.Path(path, opts.lockPath)
			locked, migrated, err := lock.Read(lockPath)
			if err != nil {
				return opErr(err)
			}
			if migrated {
				fmt.Fprintln(cmd.ErrOrStderr(), lock.MigrationWarning)
			}

			result, err := scanner.Scan(cmd.Context(), scanner.Options{Attestations: opts.attestations, SignerKey: opts.signerKey,
				Path:         path,
				Offline:      opts.offline,
				Network:      net,
				Policy:       pol,
				Reachability: opts.reachability,
			})
			if err != nil {
				return opErr(err)
			}

			got := diff.Compare(locked, result.Document).WithPolicy(pol)
			if err := writeDiff(cmd, got, opts.format, reportLocation(path, opts.lockPath)); err != nil {
				return opErr(err)
			}
			if opts.noFail || !got.TrustDrift() {
				return nil
			}
			return trustErr("trust drift detected")
		},
	}
}

func writeDiff(cmd *cobra.Command, got diff.Result, format, location string) error {
	switch strings.ToLower(format) {
	case "json":
		return diff.WriteJSON(cmd.OutOrStdout(), got)
	case "sarif", "junit", "markdown", "md":
		return finding.Write(cmd.OutOrStdout(), got.Findings("diff").WithLocation(location), format)
	case "text", "":
		diff.Write(cmd.OutOrStdout(), got)
		return nil
	default:
		return opErr(fmt.Errorf("unsupported format %q", format))
	}
}
