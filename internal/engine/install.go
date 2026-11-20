package engine

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// ReleasesRepo is the public repository that publishes engine executables. The engine
// source is private; its binaries are built and attested there.
const ReleasesRepo = "lafke/engine-releases"

// EnvVersion pins the engine version that Locate uses from the install directory.
const EnvVersion = "LAFKE_ENGINE_VERSION"

const (
	manifestName = "engine-manifest.json"
	metaName     = "engine.json"

	maxManifest = 1 << 20
	maxArchive  = 256 << 20
	maxBinary   = 512 << 20
)

// Endpoints of the release host. Tests replace them.
var (
	releasesURL = "https://github.com/" + ReleasesRepo + "/releases"
	releasesAPI = "https://api.github.com/repos/" + ReleasesRepo + "/releases"
)

// verifyAttestation checks the GitHub attestation of a downloaded file. Tests replace it.
var verifyAttestation = func(ctx context.Context, path string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return errors.New("the GitHub CLI (gh) is required to verify provenance; install it or skip verification")
	}
	out, err := exec.CommandContext(ctx, "gh", "attestation", "verify", path, "--repo", ReleasesRepo).CombinedOutput()
	if err != nil {
		return fmt.Errorf("attestation verification failed for %s: %s", filepath.Base(path), strings.TrimSpace(string(out)))
	}
	return nil
}

// Manifest describes one engine release: the engine version, the protocol it speaks and
// the sha256 of every platform archive.
type Manifest struct {
	Version  string            `json:"version"`
	Protocol int               `json:"protocol"`
	Archives map[string]string `json:"archives"`
}

// Installed is an engine in the install directory.
type Installed struct {
	Version  string `json:"version"`
	Protocol int    `json:"protocol"`
	Path     string `json:"-"`

	// Reused is set by Install when the version was already installed and nothing was
	// downloaded.
	Reused bool `json:"-"`
}

// InstallOptions controls Install.
type InstallOptions struct {
	// Version is a release tag; empty selects the newest release that speaks this
	// CLI's protocol.
	Version string

	// SkipAttestation installs without verifying provenance. The sha256 from the
	// manifest is still checked.
	SkipAttestation bool

	// Force downloads the engine again even when that version is already installed.
	Force bool

	// Token authenticates GitHub requests; empty sends none.
	Token string

	Progress func(string)
}

// InstallDir is where Install puts engines: one directory per version.
func InstallDir() string {
	root, err := os.UserCacheDir()
	if err != nil {
		root = os.TempDir()
	}
	return filepath.Join(root, "lafke", "engine")
}

func executableName() string {
	if runtime.GOOS == "windows" {
		return binaryName + ".exe"
	}
	return binaryName
}

func archiveName() string {
	name := fmt.Sprintf("%s_%s_%s", binaryName, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		return name + ".zip"
	}
	return name + ".tar.gz"
}

// List returns the installed engines that speak this CLI's protocol, newest first.
func List() []Installed {
	entries, err := os.ReadDir(InstallDir())
	if err != nil {
		return nil
	}
	var out []Installed
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(InstallDir(), e.Name())
		raw, err := os.ReadFile(filepath.Join(dir, metaName))
		if err != nil {
			continue
		}
		var meta Installed
		if json.Unmarshal(raw, &meta) != nil || meta.Protocol != ProtocolVersion || meta.Version != e.Name() {
			continue
		}
		meta.Path = filepath.Join(dir, executableName())
		if st, err := os.Stat(meta.Path); err != nil || st.IsDir() {
			continue
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return semver.Compare(out[i].Version, out[j].Version) > 0 })
	return out
}

// installedPath returns the engine Locate should use from the install directory.
func installedPath() (string, bool) {
	list := List()
	if pin := os.Getenv(EnvVersion); pin != "" {
		for _, in := range list {
			if in.Version == pin {
				return in.Path, true
			}
		}
		return "", false
	}
	if len(list) == 0 {
		return "", false
	}
	return list[0].Path, true
}

func httpGet(ctx context.Context, url, token string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("GET %s: response larger than %d bytes", url, limit)
	}
	return body, nil
}

// fetchManifest reads the manifest of a release tag; an empty tag means the latest release.
func fetchManifest(ctx context.Context, tag, token string) ([]byte, Manifest, error) {
	url := releasesURL + "/latest/download/" + manifestName
	if tag != "" {
		url = releasesURL + "/download/" + tag + "/" + manifestName
	}
	raw, err := httpGet(ctx, url, token, maxManifest)
	if err != nil {
		return nil, Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, Manifest{}, fmt.Errorf("invalid engine manifest: %w", err)
	}
	if !semver.IsValid(m.Version) || (tag != "" && m.Version != tag) {
		return nil, Manifest{}, fmt.Errorf("invalid engine manifest: version %q", m.Version)
	}
	return raw, m, nil
}

// releaseTags lists the published release tags of the engine repository, newest first.
func releaseTags(ctx context.Context, token string) ([]string, error) {
	body, err := httpGet(ctx, releasesAPI+"?per_page=30", token, maxManifest)
	if err != nil {
		return nil, err
	}
	var releases []struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("invalid release list: %w", err)
	}
	var tags []string
	for _, r := range releases {
		if !r.Draft && !r.Prerelease && semver.IsValid(r.Tag) {
			tags = append(tags, r.Tag)
		}
	}
	sort.Slice(tags, func(i, j int) bool { return semver.Compare(tags[i], tags[j]) > 0 })
	return tags, nil
}

// Remote returns the published engine releases that speak this CLI's protocol and have a
// build for this platform, newest first.
func Remote(ctx context.Context, token string) ([]Manifest, error) {
	tags, err := releaseTags(ctx, token)
	if err != nil {
		return nil, err
	}
	var out []Manifest
	for _, tag := range tags {
		_, m, err := fetchManifest(ctx, tag, token)
		if err != nil || m.Protocol != ProtocolVersion {
			continue
		}
		if _, ok := m.Archives[archiveName()]; ok {
			out = append(out, m)
		}
	}
	return out, nil
}

// resolve finds the manifest to install: the requested tag, or the newest release whose
// protocol this CLI speaks.
func resolve(ctx context.Context, opts InstallOptions) ([]byte, Manifest, error) {
	if opts.Version != "" {
		raw, m, err := fetchManifest(ctx, opts.Version, opts.Token)
		if err != nil {
			return nil, Manifest{}, err
		}
		if m.Protocol != ProtocolVersion {
			return nil, Manifest{}, fmt.Errorf("engine %s speaks protocol %d; this CLI needs %d", m.Version, m.Protocol, ProtocolVersion)
		}
		return raw, m, nil
	}
	raw, m, err := fetchManifest(ctx, "", opts.Token)
	if err == nil && m.Protocol == ProtocolVersion {
		return raw, m, nil
	}
	// The latest release may speak a newer protocol; look for the newest one that fits.
	tags, lerr := releaseTags(ctx, opts.Token)
	if lerr != nil {
		if err != nil {
			return nil, Manifest{}, err
		}
		return nil, Manifest{}, lerr
	}
	for _, tag := range tags {
		raw, m, err := fetchManifest(ctx, tag, opts.Token)
		if err == nil && m.Protocol == ProtocolVersion {
			return raw, m, nil
		}
	}
	return nil, Manifest{}, fmt.Errorf("no engine release speaks protocol %d; update lafke", ProtocolVersion)
}

// Install downloads, verifies and installs an engine and returns it.
func Install(ctx context.Context, opts InstallOptions) (Installed, error) {
	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}
	progress("resolving engine release")
	manifestRaw, m, err := resolve(ctx, opts)
	if err != nil {
		return Installed{}, err
	}
	if !opts.Force {
		for _, in := range List() {
			if in.Version != m.Version {
				continue
			}
			// Only an executable that still reports its version counts as installed.
			if v, err := checkPath(in.Path); err == nil && v == m.Version {
				in.Reused = true
				return in, nil
			}
		}
	}
	name := archiveName()
	want, ok := m.Archives[name]
	if !ok {
		return Installed{}, fmt.Errorf("engine %s has no build for %s/%s", m.Version, runtime.GOOS, runtime.GOARCH)
	}

	tmp, err := os.MkdirTemp("", "lafke-engine-")
	if err != nil {
		return Installed{}, err
	}
	defer os.RemoveAll(tmp)

	if !opts.SkipAttestation {
		progress("verifying manifest")
		path := filepath.Join(tmp, manifestName)
		if err := os.WriteFile(path, manifestRaw, 0o600); err != nil {
			return Installed{}, err
		}
		if err := verifyAttestation(ctx, path); err != nil {
			return Installed{}, err
		}
	}

	progress("downloading engine " + m.Version)
	archive, err := httpGet(ctx, releasesURL+"/download/"+m.Version+"/"+name, opts.Token, maxArchive)
	if err != nil {
		return Installed{}, err
	}
	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != want {
		return Installed{}, fmt.Errorf("checksum mismatch for %s: expected %s, got %s", name, want, got)
	}
	archivePath := filepath.Join(tmp, name)
	if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
		return Installed{}, err
	}
	if !opts.SkipAttestation {
		progress("verifying attestation")
		if err := verifyAttestation(ctx, archivePath); err != nil {
			return Installed{}, err
		}
	}

	binPath := filepath.Join(tmp, "bin-"+executableName())
	if err := extractBinary(archivePath, binPath); err != nil {
		return Installed{}, err
	}
	if !opts.SkipAttestation {
		if err := verifyAttestation(ctx, binPath); err != nil {
			return Installed{}, err
		}
	}

	dir := filepath.Join(InstallDir(), m.Version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Installed{}, err
	}
	dest := filepath.Join(dir, executableName())
	if err := replaceFile(binPath, dest); err != nil {
		return Installed{}, err
	}
	meta, err := json.Marshal(Installed{Version: m.Version, Protocol: m.Protocol})
	if err != nil {
		return Installed{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, metaName), meta, 0o644); err != nil {
		return Installed{}, err
	}

	// The installed executable has to agree with its manifest before anything uses it.
	if v, err := checkPath(dest); err != nil || v != m.Version {
		os.RemoveAll(dir)
		if err == nil {
			err = fmt.Errorf("engine reports version %s, expected %s", v, m.Version)
		}
		return Installed{}, err
	}
	return Installed{Version: m.Version, Protocol: m.Protocol, Path: dest}, nil
}

// replaceFile moves src to dest through a temporary file in dest's directory, so a
// running engine is never half-written.
func replaceFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.CreateTemp(filepath.Dir(dest), ".engine-*")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Chmod(0o755); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(out.Name(), dest)
}

// extractBinary writes the engine executable from a release archive to dest.
func extractBinary(archivePath, dest string) error {
	want := executableName()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return err
	}
	defer out.Close()
	copyLimited := func(r io.Reader) error {
		n, err := io.Copy(out, io.LimitReader(r, maxBinary+1))
		if err != nil {
			return err
		}
		if n > maxBinary {
			return errors.New("engine executable is too large")
		}
		return nil
	}

	if strings.HasSuffix(archivePath, ".zip") {
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return err
		}
		defer zr.Close()
		for _, f := range zr.File {
			if f.Name != want {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			return copyLimited(rc)
		}
		return fmt.Errorf("%s not found in archive", want)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s not found in archive", want)
		}
		if err != nil {
			return err
		}
		if h.Typeflag == tar.TypeReg && h.Name == want {
			return copyLimited(tr)
		}
	}
}

// Remove deletes installed engine versions other than keep and returns the versions removed.
func Remove(keep string) ([]string, error) {
	entries, err := os.ReadDir(InstallDir())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == keep {
			continue
		}
		if err := os.RemoveAll(filepath.Join(InstallDir(), e.Name())); err != nil {
			return removed, err
		}
		removed = append(removed, e.Name())
	}
	return removed, nil
}
