# envcheck

> **Fail fast when `.env` drifts from `.env.example`.**

`envcheck` is a small Go CLI that compares an environment file with its template. It catches the three mistakes that make a service fail at startup instead of at review time:

- a key that exists in the template but not in the real environment;
- a key that exists but is empty;
- a key that only exists locally (a warning by default, an error with `-strict`).

No dependencies outside the Go standard library.

## Why this exists

Every project eventually has a `.env.example` that says `DATABASE_URL=` and a `.env` that forgot it. The service starts, panics on the first request, and the stack trace points somewhere else. `envcheck` turns that into a one-line failure before the process starts.

## Install

```sh
go install github.com/melinoeluna4/envcheck@latest
```

## Usage

```sh
envcheck
```

```text
envcheck: .env against .env.example
missing (1): DATABASE_URL
empty (1): API_TOKEN
extra (1): DEBUG
FAIL
```

Use another pair of files, JSON output, or treat extra keys as errors:

```sh
envcheck -env .env.local -template .env.example
envcheck -json
envcheck -strict
```

### Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-env` | `.env` | environment file to validate |
| `-template` | `.env.example` | template with the required keys |
| `-strict` | `false` | make extra keys fail too |
| `-json` | `false` | print a JSON report |

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | every template key exists and has a value |
| `1` | missing keys, empty values, or extra keys with `-strict` |
| `2` | usage or I/O error |

## How it works

1. Read both files as `KEY=VALUE` lines, skipping blanks and `#` comments.
2. Require every key from the template in the environment file.
3. Report missing, empty and extra keys in sorted order.
4. Exit non-zero so a script, a container entrypoint or CI can stop.

The parser handles `export KEY=VALUE` and quoted values without trying to be a full shell parser.

## Development

```sh
go test ./...
go vet ./...
go build ./...
```

CI runs the same three commands on every push and pull request.

---

Part of the [Melinoe Luna portfolio](https://melinoeluna4.github.io/proyectos/envcheck/) · Go 1.27 · standard library only.
