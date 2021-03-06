package zipfs

import (
	"github.com/jasonmoo/go-fuse/fuse"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

var _ = log.Printf

const testTtl = 100 * time.Millisecond

func setupMzfs(t *testing.T) (mountPoint string, cleanup func()) {
	fs := NewMultiZipFs()
	mountPoint, _ = ioutil.TempDir("", "")
	nfs := fuse.NewPathNodeFs(fs, nil)
	state, _, err := fuse.MountNodeFileSystem(mountPoint, nfs, &fuse.FileSystemOptions{
		EntryTimeout:    testTtl,
		AttrTimeout:     testTtl,
		NegativeTimeout: 0.0,
	})
	if err != nil {
		t.Fatalf("MountNodeFileSystem failed: %v", err)
	}
	state.Debug = fuse.VerboseTest()
	go state.Loop()

	return mountPoint, func() {
		state.Unmount()
		os.RemoveAll(mountPoint)
	}
}

func TestMultiZipReadonly(t *testing.T) {
	mountPoint, cleanup := setupMzfs(t)
	defer cleanup()

	_, err := os.Create(mountPoint + "/random")
	if err == nil {
		t.Error("Must fail writing in root.")
	}

	_, err = os.OpenFile(mountPoint+"/config/zipmount", os.O_WRONLY, 0)
	if err == nil {
		t.Error("Must fail without O_CREATE")
	}
}

func TestMultiZipFs(t *testing.T) {
	mountPoint, cleanup := setupMzfs(t)
	defer cleanup()

	zipFile := testZipFile()

	entries, err := ioutil.ReadDir(mountPoint)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 1 || string(entries[0].Name()) != "config" {
		t.Errorf("wrong names return. %v", entries)
	}

	err = os.Symlink(zipFile, mountPoint+"/config/zipmount")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	fi, err := os.Lstat(mountPoint + "/zipmount")
	if !fi.IsDir() {
		t.Errorf("Expect directory at /zipmount")
	}

	entries, err = ioutil.ReadDir(mountPoint)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 2 {
		t.Error("Expect 2 entries", entries)
	}

	val, err := os.Readlink(mountPoint + "/config/zipmount")
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if val != zipFile {
		t.Errorf("expected %v got %v", zipFile, val)
	}

	fi, err = os.Lstat(mountPoint + "/zipmount")
	if err != nil {
		t.Fatalf("Lstat failed: %v", err)
	}
	if !fi.IsDir() {
		t.Fatal("expect directory for /zipmount, got %v", fi)
	}

	// Check that zipfs itself works.
	fi, err = os.Stat(mountPoint + "/zipmount/subdir")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if !fi.IsDir() {
		t.Error("directory type", fi)
	}

	// Removing the config dir unmount
	err = os.Remove(mountPoint + "/config/zipmount")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	fi, err = os.Stat(mountPoint + "/zipmount")
	if err == nil {
		t.Errorf("stat should fail after unmount, got %#v", fi)
	}
}
