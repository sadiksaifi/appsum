# appsum

See CPU and memory usage for apps and process trees, with a total followed by every descendant. macOS reports physical footprint; Linux reports proportional set size (PSS).

<img src="https://github.com/user-attachments/assets/28edc867-0f18-41f3-82e2-40dedea1ce37" alt="App totals and a nested process tree showing CPU usage and physical memory footprint" width="100%">

## Install

Supports macOS and Linux on x86_64 and arm64. No C compiler or Apple developer tools are required.

```sh
CGO_ENABLED=0 go install github.com/sadiksaifi/appsum/cmd/appsum@latest
```

Linux builds are tested on both architectures. Intel Mac builds are cross-compiled but not yet tested on Intel hardware.

## Usage

```sh
appsum
appsum "<app-name/process-name>"
appsum <pid>
appsum "<app-name/process-name>" --watch
appsum "<app-name/process-name>" --json
```

Exact names take priority over partial matches. CPU 100% means one busy core. macOS footprint and Linux PSS are different accounting metrics; RSS is never substituted. JSON identifies the metric with `memory_metric` and reports values in `memory_bytes`.

### Flags

| Flag | Default | What it does |
| --- | --- | --- |
| `-w`, `--watch` | Off | Refresh in place. Stream snapshots when piped or when the full report cannot fit the terminal height. |
| `--json` | Off | Output the complete tree as JSON. With `--watch`, emit one JSON object per line. |
| `-v`, `--verbose` | Off | Show untruncated command lines, or executable paths when the command line is unavailable. |
| `--interval <duration>` | `1s` | Set the CPU sampling interval, such as `500ms`. Minimum: `100ms`. |
| `--depth <levels>` | `0` | Limit displayed process levels; `0` shows all. Totals and JSON always include every descendant. |
| `-h`, `--help` | — | Show usage and examples. |
| `--version` | — | Show the build version. |
