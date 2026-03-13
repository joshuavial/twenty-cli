package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var updateGolden = flag.Bool("update", false, "update golden files")

type Harness struct {
	t      *testing.T
	binary string
	root   string
}

type RunOptions struct {
	Args            []string
	Env             map[string]string
	WorkingDirFiles map[string]string
	HomeFiles       map[string]string
}

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

var (
	buildOnce   sync.Once
	buildBinary string
	buildErr    error
)

func newHarness(t *testing.T) *Harness {
	t.Helper()

	root := repoRoot(t)
	binary, err := compiledBinary(root)
	if err != nil {
		t.Fatalf("compiledBinary() error = %v", err)
	}

	return &Harness{
		t:      t,
		binary: binary,
		root:   root,
	}
}

func (h *Harness) Run(opts RunOptions) Result {
	h.t.Helper()

	homeDir := h.t.TempDir()
	workDir := h.t.TempDir()

	writeFiles(h.t, homeDir, opts.HomeFiles)
	writeFiles(h.t, workDir, opts.WorkingDirFiles)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.binary, opts.Args...)
	cmd.Dir = workDir
	cmd.Env = commandEnv(homeDir, opts.Env)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	if err == nil {
		return result
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result
	}

	h.t.Fatalf("command run error = %v", err)
	return Result{}
}

func (h *Harness) AssertJSONSnapshot(name string, result Result) {
	h.t.Helper()

	if strings.TrimSpace(result.Stderr) != "" {
		h.t.Fatalf("stderr = %q, want empty", result.Stderr)
	}

	got := prettyJSON(h.t, result.Stdout)
	snapshot := filepath.Join(h.root, "test", "e2e", "testdata", name+".golden.json")

	if *updateGolden || os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(snapshot, []byte(got), 0o600); err != nil {
			h.t.Fatalf("WriteFile(%s) error = %v", snapshot, err)
		}
	}

	wantBytes, err := os.ReadFile(snapshot)
	if err != nil {
		h.t.Fatalf("ReadFile(%s) error = %v", snapshot, err)
	}

	want := string(wantBytes)
	if got != want {
		h.t.Fatalf("snapshot %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func compiledBinary(root string) (string, error) {
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "twenty-cli-bin-*")
		if err != nil {
			buildErr = err
			return
		}

		buildBinary = filepath.Join(dir, "twenty")
		cmd := exec.Command("go", "build", "-o", buildBinary, "./cmd/twenty")
		cmd.Dir = root
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("go build: %w: %s", err, strings.TrimSpace(string(output)))
		}
	})

	return buildBinary, buildErr
}

func commandEnv(homeDir string, extra map[string]string) []string {
	env := make([]string, 0, len(extra)+4)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "TWENTY_API_KEY=") || strings.HasPrefix(entry, "TWENTY_BASE_URL=") || strings.HasPrefix(entry, "HOME=") {
			continue
		}

		env = append(env, entry)
	}

	env = append(env, "HOME="+homeDir)
	for key, value := range extra {
		env = append(env, key+"="+value)
	}

	return env
}

func prettyJSON(t *testing.T, raw string) string {
	t.Helper()

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; raw = %s", err, raw)
	}

	formatted, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}

	return string(formatted) + "\n"
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for relativePath, contents := range files {
		fullPath := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(contents), 0o600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", fullPath, err)
		}
	}
}
