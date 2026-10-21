# appsum

See an app's CPU usage and physical memory footprint, with a total followed by its full process tree. For Apple Silicon Macs. Building requires Apple's command-line tools.

<img src="https://github.com/user-attachments/assets/1570acb5-c546-4fd5-bddf-625cb4aaebfc" alt="App totals and a nested process tree showing CPU usage and physical memory footprint" width="100%">

## Install

```sh
go install github.com/sadiksaifi/appsum/cmd/appsum@latest
```

## Usage

```sh
appsum
appsum "<app-name/process-name>"
appsum <pid>
appsum "<app-name/process-name>" --watch
appsum "<app-name/process-name>" --json
```

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
