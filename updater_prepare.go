//go:build !js

package mygo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/egoist/mygo/internal/update"
)

// PreparedUpdate owns a verified, unpacked update without changing the installed
// app. Install commits it after the application's normal dirty-content barrier;
// Close discards an uncommitted preparation. Both methods are goroutine-safe.
type PreparedUpdate struct {
	mu                        sync.Mutex
	work, dir, target         string
	installed, staged, closed bool
}

// UpdatePreparationOptions reports verified downloads separately from archive
// extraction. Callbacks run on the goroutine calling Prepare, not the UI thread.
type UpdatePreparationOptions struct {
	Progress  func(downloaded, total int64)
	Preparing func()
}

func validatePreparedUpdate(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("the update is empty")
	}
	if runtime.GOOS == "darwin" && (len(entries) != 1 || !entries[0].IsDir() || !strings.HasSuffix(entries[0].Name(), ".app")) {
		return errors.New("the update holds no app bundle")
	}
	return nil
}

// Prepare downloads, verifies and extracts the update without replacing or
// staging the installed app. The caller must Close the result when abandoning it.
func (up *Update) Prepare(ctx context.Context, options UpdatePreparationOptions) (*PreparedUpdate, error) {
	target, err := installTarget()
	if err != nil {
		return nil, err
	}
	get := httpGet
	if up.source != nil {
		get = up.source.Open
	}
	prepared, err := prepareUpdate(ctx, get, up.manifest, up.key, up.current, target, options.Progress, options.Preparing)
	if err != nil {
		return nil, fmt.Errorf("mygo: preparing the update: %w", err)
	}
	return prepared, nil
}

// Install commits the prepared app. On macOS the existing helper waits for this
// process to exit before swapping bundles; App.Relaunch requests reopening it.
func (prepared *PreparedUpdate) Install(ctx context.Context) error {
	if err := prepared.install(ctx); err != nil {
		return fmt.Errorf("mygo: installing the update: %w", err)
	}
	if runtime.GOOS == "linux" {
		if err := update.RefreshDesktopEntry(prepared.target, filepath.Base(startExe)); err != nil {
			log.Printf("mygo: registering the desktop entry of the update: %v", err)
		}
	}
	return nil
}

func (prepared *PreparedUpdate) install(ctx context.Context) error {
	if prepared == nil {
		return errors.New("update is not prepared")
	}
	prepared.mu.Lock()
	defer prepared.mu.Unlock()
	if prepared.installed {
		return nil
	}
	if prepared.closed || prepared.work == "" {
		return errors.New("update preparation is closed")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		if err := stageBundle(prepared.work, prepared.dir, prepared.target); err != nil {
			return err
		}
		prepared.staged = true
	} else if err := swapFiles(prepared.dir, prepared.target); err != nil {
		return err
	}
	prepared.installed = true
	return nil
}

// Close removes only this preparation's temporary files. An installed macOS
// preparation stays owned by the existing after-exit helper.
func (prepared *PreparedUpdate) Close() error {
	if prepared == nil {
		return nil
	}
	prepared.mu.Lock()
	defer prepared.mu.Unlock()
	if prepared.closed {
		return nil
	}
	prepared.closed = true
	if prepared.staged || prepared.work == "" {
		return nil
	}
	return os.RemoveAll(prepared.work)
}
