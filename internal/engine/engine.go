// Package engine runs the Lafke analysis engine, a separate executable that the CLI
// starts for source-code analysis. Requests and responses are JSON; see the engine's
// protocol version below.
package engine

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lafke/lafke/internal/finding"
)

// ProtocolVersion is the engine protocol this CLI speaks.
const ProtocolVersion = 1

// EnvPath names an environment variable that overrides where the engine is found.
const EnvPath = "LAFKE_ENGINE"

const binaryName = "lafke-engine"

// ErrNotFound is returned when no engine executable can be located.
var ErrNotFound = errors.New("lafke-engine not found: install it next to the lafke executable, put it on PATH, or set " + EnvPath)

// Options tunes the analysis.
type Options struct {
	// AssumeParams treats string parameters of Go functions that nothing calls
	// directly as untrusted; it finds more, with more false positives.
	AssumeParams bool

	// Progress receives coarse analysis stages; nil disables updates.
	Progress func(string)
}

// Symbol names a vulnerable function for call-graph reachability.
type Symbol struct {
	ID   string `json:"id"`
	Name string `json:"symbol"`
}

type request struct {
	Root         string   `json:"root,omitempty"`
	AssumeParams bool     `json:"assume_params,omitempty"`
	Entries      []string `json:"entries,omitempty"`
	Symbols      []Symbol `json:"symbols,omitempty"`
	Files        []file   `json:"files,omitempty"`
}

type file struct {
	Name    string `json:"name"`
	Content []byte `json:"content"`
}

// Locate returns the path of the engine executable.
func Locate() (string, error) {
	if p := os.Getenv(EnvPath); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("%s: %w", EnvPath, err)
		}
		return p, nil
	}
	name := binaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		candidate := filepath.Join(filepath.Dir(exe), name)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	return "", ErrNotFound
}

// Available reports whether an engine executable can be located.
func Available() bool {
	_, err := Locate()
	return err == nil
}

// call runs one engine command and decodes its JSON response into out.
func call(command string, req request, progress func(string), out any) error {
	path, err := Locate()
	if err != nil {
		return err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	cmd := exec.Command(path, command)
	cmd.Stdin = bytes.NewReader(body)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start engine: %w", err)
	}
	var diagnostics []string
	sc := bufio.NewScanner(stderr)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if stage, ok := strings.CutPrefix(line, "progress: "); ok {
			if progress != nil {
				progress(stage)
			}
			continue
		}
		diagnostics = append(diagnostics, line)
	}
	if err := cmd.Wait(); err != nil {
		if len(diagnostics) > 0 {
			return errors.New(strings.TrimPrefix(strings.Join(diagnostics, "\n"), "lafke-engine: "))
		}
		return fmt.Errorf("engine failed: %w", err)
	}
	if err := json.Unmarshal(stdout.Bytes(), out); err != nil {
		return fmt.Errorf("invalid engine response: %w", err)
	}
	return nil
}

// CheckVersion verifies that the engine speaks this CLI's protocol and returns its
// version string.
func CheckVersion() (string, error) {
	path, err := Locate()
	if err != nil {
		return "", err
	}
	raw, err := exec.Command(path, "version").Output()
	if err != nil {
		return "", fmt.Errorf("run engine: %w", err)
	}
	var v struct {
		Protocol int    `json:"protocol"`
		Version  string `json:"version"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("invalid engine response: %w", err)
	}
	if v.Protocol != ProtocolVersion {
		return v.Version, fmt.Errorf("engine protocol %d is not supported (this CLI needs %d); update lafke and lafke-engine together", v.Protocol, ProtocolVersion)
	}
	return v.Version, nil
}

// Analyze scans the source under root and returns findings sorted by severity.
func Analyze(root string) ([]finding.Finding, error) {
	return AnalyzeWith(root, Options{})
}

// AnalyzeWith is Analyze with options.
func AnalyzeWith(root string, opts Options) ([]finding.Finding, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var found []finding.Finding
	if err := call("analyze", request{Root: abs, AssumeParams: opts.AssumeParams}, opts.Progress, &found); err != nil {
		return nil, err
	}
	return found, nil
}

// SecretContent scans one file's content for credentials.
func SecretContent(rel string, src []byte) []finding.Finding {
	return SecretContents([]string{rel}, [][]byte{src})[0]
}

// SecretContents scans several files in one engine run; the result matches the input
// order. Files that cannot be scanned yield no findings.
func SecretContents(names []string, contents [][]byte) [][]finding.Finding {
	out := make([][]finding.Finding, len(names))
	req := request{}
	for i, n := range names {
		req.Files = append(req.Files, file{Name: n, Content: contents[i]})
	}
	var res [][]finding.Finding
	if err := call("secrets", req, nil, &res); err != nil || len(res) != len(names) {
		return out
	}
	return res
}

// Calls builds the call graph under root and reports which symbols the entries reach.
func Calls(root string, entries []string, symbols []Symbol) (finding.Report, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return finding.New("reachability", nil), err
	}
	var report finding.Report
	if err := call("calls", request{Root: abs, Entries: entries, Symbols: symbols}, nil, &report); err != nil {
		return finding.New("reachability", nil), err
	}
	return report, nil
}
