# appsum

![App totals and a nested process tree showing CPU usage and physical memory footprint](https://github.com/user-attachments/assets/e5293b25-0d47-4863-aa45-aea786ad7852)

See an app's CPU usage and physical memory footprint, with a total followed by its full process tree.
For Apple Silicon Macs. Building requires Apple's command-line tools.
```sh
go install github.com/sadiksaifi/appsum/cmd/appsum@latest
appsum "<app-name/process-name>" --watch
```
