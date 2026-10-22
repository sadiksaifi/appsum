package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
)

// Set by GoReleaser; go install supplies the module version instead.
var version string

func currentVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func upgrade(ctx context.Context, out io.Writer) error {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return err
	}
	path, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}
	return upgradeBinary(ctx, out, currentVersion(), path, updater)
}

func upgradeBinary(ctx context.Context, out io.Writer, current, path string, updater *selfupdate.Updater) error {
	// ExecutablePath resolves bin/ and opt/ symlinks to the installed keg.
	keg := filepath.Dir(filepath.Dir(path))
	rack := filepath.Dir(keg)
	cellar := filepath.Dir(rack)
	if filepath.Base(filepath.Dir(path)) == "bin" && filepath.Base(rack) == "appsum" && filepath.Base(cellar) == "Cellar" {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		if _, err := fmt.Fprintln(out, "Homebrew installation detected; upgrading with Homebrew..."); err != nil {
			return err
		}
		// Use the owning Homebrew, even when PATH selects another installation.
		brew := filepath.Join(filepath.Dir(cellar), "bin", "brew")
		cmd := exec.CommandContext(ctx, brew, "upgrade", "sadiksaifi/tap/appsum")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, out, out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("brew upgrade failed (run %s upgrade sadiksaifi/tap/appsum to retry): %w", brew, err)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	installed, err := semver.NewVersion(current)
	if err != nil {
		return fmt.Errorf("cannot upgrade a development build; install an official release first")
	}
	if _, err := fmt.Fprintln(out, "Checking for a stable release..."); err != nil {
		return err
	}
	release, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug("sadiksaifi", "appsum"))
	if err != nil {
		return fmt.Errorf("check release: %w", err)
	}
	if !found {
		return fmt.Errorf("no compatible stable release found")
	}
	latest, err := semver.NewVersion(release.Version())
	if err != nil || latest.Prerelease() != "" || latest.Metadata() != "" {
		return fmt.Errorf("release %q is not a stable version", release.Version())
	}
	if !latest.GreaterThan(installed) {
		_, err = fmt.Fprintf(out, "appsum %s is already up to date.\n", current)
		return err
	}
	if _, err := fmt.Fprintf(out, "Upgrading to v%s...\n", release.Version()); err != nil {
		return err
	}
	if err := updater.UpdateTo(ctx, release, path); err != nil {
		return fmt.Errorf("upgrade %s (the installation directory must be writable): %w", path, err)
	}
	_, err = fmt.Fprintf(out, "Updated appsum to v%s.\n", release.Version())
	return err
}
