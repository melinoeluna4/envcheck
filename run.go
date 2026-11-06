package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const usage = `envcheck compares an environment file with its template.

Usage:
  envcheck [flags]

Flags:
  -env string       environment file (default ".env")
  -template string  template file (default ".env.example")
  -strict           treat extra keys as errors
  -json             print a JSON report

Exit codes:
  0  every template key exists and has a value
  1  missing keys, empty values, or extra keys with -strict
  2  usage or I/O error
`

type report struct {
	Env      string   `json:"env"`
	Template string   `json:"template"`
	Missing  []string `json:"missing"`
	Empty    []string `json:"empty"`
	Extra    []string `json:"extra"`
	OK       bool     `json:"ok"`
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("envcheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprint(stderr, usage)
	}

	envPath := flags.String("env", ".env", "environment file")
	templatePath := flags.String("template", ".env.example", "template file")
	strict := flags.Bool("strict", false, "treat extra keys as errors")
	asJSON := flags.Bool("json", false, "print a JSON report")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}

		return 2
	}

	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "envcheck: unexpected arguments: %s\n", strings.Join(flags.Args(), " "))
		return 2
	}

	env, err := readEnvFile(*envPath)
	if err != nil {
		fmt.Fprintf(stderr, "envcheck: %v\n", err)
		return 2
	}

	template, err := readEnvFile(*templatePath)
	if err != nil {
		fmt.Fprintf(stderr, "envcheck: %v\n", err)
		return 2
	}

	result := compare(*envPath, *templatePath, env, template)
	result.OK = len(result.Missing) == 0 &&
		len(result.Empty) == 0 &&
		(!*strict || len(result.Extra) == 0)

	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(stderr, "envcheck: %v\n", err)
			return 2
		}
	} else {
		printReport(stdout, result, *strict)
	}

	if !result.OK {
		return 1
	}

	return 0
}

func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	values, err := parseEnv(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return values, nil
}

func parseEnv(reader io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	line := 0
	for scanner.Scan() {
		line++

		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}

		raw = strings.TrimPrefix(raw, "export ")
		key, value, ok := strings.Cut(raw, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", line)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", line)
		}

		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	return values, nil
}

func compare(envPath, templatePath string, env, template map[string]string) report {
	result := report{
		Env:      envPath,
		Template: templatePath,
		Missing:  []string{},
		Empty:    []string{},
		Extra:    []string{},
	}

	for key := range template {
		value, ok := env[key]
		if !ok {
			result.Missing = append(result.Missing, key)
			continue
		}

		if strings.TrimSpace(value) == "" {
			result.Empty = append(result.Empty, key)
		}
	}

	for key := range env {
		if _, ok := template[key]; !ok {
			result.Extra = append(result.Extra, key)
		}
	}

	sort.Strings(result.Missing)
	sort.Strings(result.Empty)
	sort.Strings(result.Extra)

	return result
}

func printReport(writer io.Writer, result report, strict bool) {
	fmt.Fprintf(writer, "envcheck: %s against %s\n", result.Env, result.Template)

	if len(result.Missing) == 0 && len(result.Empty) == 0 && len(result.Extra) == 0 {
		fmt.Fprintln(writer, "OK")
		return
	}

	if len(result.Missing) > 0 {
		fmt.Fprintf(writer, "missing (%d): %s\n", len(result.Missing), strings.Join(result.Missing, ", "))
	}

	if len(result.Empty) > 0 {
		fmt.Fprintf(writer, "empty (%d): %s\n", len(result.Empty), strings.Join(result.Empty, ", "))
	}

	if len(result.Extra) > 0 {
		fmt.Fprintf(writer, "extra (%d): %s\n", len(result.Extra), strings.Join(result.Extra, ", "))
	}

	if len(result.Missing) == 0 && len(result.Empty) == 0 && !strict {
		fmt.Fprintln(writer, "OK (extra keys ignored; use -strict to fail on them)")
		return
	}

	fmt.Fprintln(writer, "FAIL")
}
