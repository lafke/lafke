package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lafke/cli/internal/httpx"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const engineBinaryName = "lafke-engine"

// executablePath is replaced in tests, which must not remove the test binary.
var executablePath = os.Executable

type uninstallOptions struct {
	yes       bool
	dryRun    bool
	keepCache bool
}

func newUninstallCommand() *cobra.Command {
	uo := &uninstallOptions{}
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove lafke, its analysis engine and its cache from this machine",
		Long: `Remove the lafke executable that is running, the lafke-engine executable next to
it, and the on-disk response cache. Project files such as lafke.lock and
.lafke/ are never touched. An engine found through LAFKE_ENGINE or PATH is left
alone, because the installer did not put it there.

Without --yes the command lists what it will remove and asks first. If lafke was
installed by a package manager, uninstall it with that package manager instead.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(cmd, uo)
		},
	}
	cmd.Flags().BoolVarP(&uo.yes, "yes", "y", false, "do not ask for confirmation")
	cmd.Flags().BoolVar(&uo.dryRun, "dry-run", false, "list what would be removed and change nothing")
	cmd.Flags().BoolVar(&uo.keepCache, "keep-cache", false, "keep the on-disk response cache")
	return cmd
}

// uninstallFiles returns the executables the installer put in the directory of exe.
func uninstallFiles(exe string) []string {
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	files := []string{exe}
	engine := filepath.Join(filepath.Dir(exe), engineBinaryName+filepath.Ext(exe))
	if st, err := os.Stat(engine); err == nil && !st.IsDir() {
		files = append(files, engine)
	}
	return files
}

func runUninstall(cmd *cobra.Command, uo *uninstallOptions) error {
	out := cmd.OutOrStdout()
	exe, err := executablePath()
	if err != nil {
		return opErr(fmt.Errorf("locate the lafke executable: %w", err))
	}
	files := uninstallFiles(exe)
	if strings.Contains(filepath.ToSlash(files[0]), "/go-build") {
		return opErr(errors.New("this is a temporary `go run` build; there is nothing installed to remove"))
	}

	fmt.Fprintln(out, "This will remove:")
	for _, f := range files {
		fmt.Fprintf(out, "  %s\n", f)
	}
	if !uo.keepCache {
		fmt.Fprintf(out, "  %s (cached responses)\n", httpx.CacheDir())
	}
	if uo.dryRun {
		return nil
	}

	if !uo.yes {
		if !isTerminalInput(cmd.InOrStdin()) {
			return opErr(errors.New("refusing to uninstall without confirmation; pass --yes"))
		}
		fmt.Fprint(out, "Continue? [y/N] ")
		line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
		if answer := strings.ToLower(strings.TrimSpace(line)); answer != "y" && answer != "yes" {
			fmt.Fprintln(out, "aborted")
			return nil
		}
	}

	var failed []string
	if !uo.keepCache {
		if _, err := httpx.Clear(); err != nil {
			failed = append(failed, fmt.Sprintf("cache: %v", err))
		} else if err := os.Remove(httpx.CacheDir()); err != nil && !errors.Is(err, fs.ErrNotExist) {
			// A directory that still holds foreign files stays; that is not a failure.
			fmt.Fprintf(out, "kept %s: not empty\n", httpx.CacheDir())
		}
	}
	// The engine goes first so a failure leaves a working lafke to retry with.
	for i := len(files) - 1; i >= 0; i-- {
		if err := removeExecutable(files[i]); err != nil {
			failed = append(failed, err.Error())
			continue
		}
		fmt.Fprintf(out, "removed %s\n", files[i])
	}
	if len(failed) > 0 {
		return opErr(fmt.Errorf("uninstall incomplete:\n  %s", strings.Join(failed, "\n  ")))
	}
	return nil
}

// removeExecutable deletes path. Windows refuses to delete a running executable but
// allows renaming it, so the file is moved aside there for the user to delete.
func removeExecutable(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if runtime.GOOS == "windows" {
		aside := path + ".old"
		if rerr := os.Rename(path, aside); rerr == nil {
			return fmt.Errorf("%s is in use; renamed to %s, delete it after this command exits", path, aside)
		}
	}
	return err
}

func isTerminalInput(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
