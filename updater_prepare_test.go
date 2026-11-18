//go:build !js

package mygo

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/update"
)

// A downloaded update must not replace the app before dirty-content approval.
// Cancellation and signature failure must leave the current app usable and
// release only the preparation's files, without launching an install helper.
func TestUpdatePreparationPreservesInstalledAppUntilCommit(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "Existing.app")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(target, "main")
	if err := os.WriteFile(installed, []byte("current app"), 0755); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	entries := []string{"main"}
	if runtime.GOOS == "darwin" {
		entries = []string{"New.app"}
		if err := os.Mkdir(filepath.Join(source, entries[0]), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(source, entries[0], "main"), []byte("new app"), 0755); err != nil {
			t.Fatal(err)
		}
	} else if err := os.WriteFile(filepath.Join(source, "main"), []byte("new app"), 0755); err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	if err := update.WriteArchive(&archive, source, entries); err != nil {
		t.Fatal(err)
	}
	public, secret, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive.Bytes())
	manifest := update.Manifest{Version: "2.0.0", Size: int64(archive.Len()), URL: "memory:update", Signature: update.Sign(secret, sum[:])}
	get := func(context.Context, string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(archive.Bytes())), nil
	}
	key := base64.StdEncoding.EncodeToString(public)
	prepared, err := prepareUpdate(context.Background(), get, manifest, key, "1.0.0", target, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = prepared.Close() })
	assertInstalled := func() {
		t.Helper()
		data, err := os.ReadFile(installed)
		if err != nil || string(data) != "current app" {
			t.Fatal("uncommitted preparation changed the installed app", err)
		}
		if pendingUpdate.Load() != nil {
			t.Fatal("preparation started an after-exit install helper")
		}
	}
	assertInstalled()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := prepared.Install(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled commit did not remain canceled", err)
	}
	assertInstalled()
	if err := prepared.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(prepared.work); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("discarded preparation retained its downloaded files", err)
	}
	manifest.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	if rejected, err := prepareUpdate(context.Background(), get, manifest, key, "1.0.0", target, nil, nil); err == nil || rejected != nil {
		t.Fatal("untrusted archive obtained commit authority")
	}
	assertInstalled()
	files, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Name() != filepath.Base(target) {
			t.Fatal("failed preparation leaked staging data", file.Name())
		}
	}
}
