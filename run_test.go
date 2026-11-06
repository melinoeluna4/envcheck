package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseEnv(t *testing.T) {
	t.Parallel()

	values, err := parseEnv(strings.NewReader("A=1\n# comment\nexport B='two'\nC=\n"))
	if err != nil {
		t.Fatalf("parseEnv returned an error: %v", err)
	}

	want := map[string]string{"A": "1", "B": "two", "C": ""}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("parseEnv = %#v, want %#v", values, want)
	}
}

func TestParseEnvRejectsInvalidLine(t *testing.T) {
	t.Parallel()

	if _, err := parseEnv(strings.NewReader("NOPE\n")); err == nil {
		t.Fatal("parseEnv accepted a line without =")
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()

	result := compare(
		".env",
		".env.example",
		map[string]string{"A": "1", "B": "", "EXTRA": "x"},
		map[string]string{"A": "", "B": "", "C": ""},
	)

	if !reflect.DeepEqual(result.Missing, []string{"C"}) {
		t.Fatalf("missing = %#v, want [C]", result.Missing)
	}
	if !reflect.DeepEqual(result.Empty, []string{"B"}) {
		t.Fatalf("empty = %#v, want [B]", result.Empty)
	}
	if !reflect.DeepEqual(result.Extra, []string{"EXTRA"}) {
		t.Fatalf("extra = %#v, want [EXTRA]", result.Extra)
	}
}

func TestRunOK(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", "DATABASE_URL=postgres://localhost/app\nTOKEN=secret\n")
	templatePath := writeFile(t, dir, ".env.example", "DATABASE_URL=\nTOKEN=\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-env", envPath, "-template", templatePath}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Fatalf("stdout = %q, want OK", stdout.String())
	}
}

func TestRunFailsOnMissingAndEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", "TOKEN=\nEXTRA=1\n")
	templatePath := writeFile(t, dir, ".env.example", "DATABASE_URL=\nTOKEN=\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-env", envPath, "-template", templatePath}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "missing (1): DATABASE_URL") {
		t.Fatalf("stdout = %q, want missing DATABASE_URL", output)
	}
	if !strings.Contains(output, "empty (1): TOKEN") {
		t.Fatalf("stdout = %q, want empty TOKEN", output)
	}
	if !strings.Contains(output, "extra (1): EXTRA") {
		t.Fatalf("stdout = %q, want extra EXTRA", output)
	}
	if !strings.Contains(output, "FAIL") {
		t.Fatalf("stdout = %q, want FAIL", output)
	}
}

func TestRunStrictExtra(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", "TOKEN=secret\nDEBUG=1\n")
	templatePath := writeFile(t, dir, ".env.example", "TOKEN=\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-env", envPath, "-template", templatePath, "-strict"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "extra (1): DEBUG") {
		t.Fatalf("stdout = %q, want extra DEBUG", stdout.String())
	}
}

func TestRunJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", "TOKEN=secret\n")
	templatePath := writeFile(t, dir, ".env.example", "TOKEN=\nDATABASE_URL=\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-env", envPath, "-template", templatePath, "-json"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}

	var result report
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v; output = %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("JSON report says OK, want failure")
	}
	if !reflect.DeepEqual(result.Missing, []string{"DATABASE_URL"}) {
		t.Fatalf("missing = %#v, want [DATABASE_URL]", result.Missing)
	}
}

func TestRunMissingTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", "TOKEN=secret\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-env", envPath, "-template", filepath.Join(dir, "missing")}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "read") {
		t.Fatalf("stderr = %q, want a read error", stderr.String())
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%s): %v", path, err)
	}

	return path
}
