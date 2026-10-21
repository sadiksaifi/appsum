# appsum

See an app's CPU usage and physical memory footprint, with a total followed by its full process tree.
For Apple Silicon Macs. Building requires Apple's command-line tools.

```sh
go install github.com/sadiksaifi/appsum/cmd/appsum@latest
appsum asklog
appsum asklog --watch
```
