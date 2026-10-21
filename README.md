# appsum

<img src="https://github.com/user-attachments/assets/b25e939b-6305-48b1-89bd-2e6f439aafc9" alt="App totals and a nested process tree showing CPU usage and physical memory footprint" width="100%">

See an app's CPU usage and physical memory footprint, with a total followed by its full process tree.
For Apple Silicon Macs. Building requires Apple's command-line tools.

```sh
go install github.com/sadiksaifi/appsum/cmd/appsum@latest
appsum "<app-name/process-name>" --watch
```

## Flags

| Flag | Default | What it does |
| --- | --- | --- |
| `-w`, `--watch` | Off | Refresh in place. Stream snapshots when piped or when the full report cannot fit the terminal height. |
| `--json` | Off | Output the complete tree as JSON. With `--watch`, emit one JSON object per line. |
| `-v`, `--verbose` | Off | Show untruncated command lines, or executable paths when the command line is unavailable. |
| `--interval <duration>` | `1s` | Set the CPU sampling interval, such as `500ms`. Minimum: `100ms`. |
| `--depth <levels>` | `0` | Limit displayed process levels; `0` shows all. Totals and JSON always include every descendant. |
| `-h`, `--help` | — | Show usage and examples. |
| `--version` | — | Show the build version. |
