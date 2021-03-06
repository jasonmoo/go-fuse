package unionfs

import (
	"fmt"
	"github.com/jasonmoo/go-fuse/fuse"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

var _ = fmt.Print
var _ = log.Print

const entryTtl = 100 * time.Millisecond

var testAOpts = AutoUnionFsOptions{
	UnionFsOptions: testOpts,
	FileSystemOptions: fuse.FileSystemOptions{
		EntryTimeout:    entryTtl,
		AttrTimeout:     entryTtl,
		NegativeTimeout: 0,
	},
	HideReadonly: true,
}

func WriteFile(t *testing.T, name string, contents string) {
	err := ioutil.WriteFile(name, []byte(contents), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func setup(t *testing.T) (workdir string, cleanup func()) {
	wd, _ := ioutil.TempDir("", "")
	err := os.Mkdir(wd+"/mnt", 0700)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	err = os.Mkdir(wd+"/store", 0700)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	os.Mkdir(wd+"/ro", 0700)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	WriteFile(t, wd+"/ro/file1", "file1")
	WriteFile(t, wd+"/ro/file2", "file2")

	fs := NewAutoUnionFs(wd+"/store", testAOpts)

	nfs := fuse.NewPathNodeFs(fs, nil)
	state, conn, err := fuse.MountNodeFileSystem(wd+"/mnt", nfs, &testAOpts.FileSystemOptions)
	if err != nil {
		t.Fatalf("MountNodeFileSystem failed: %v", err)
	}
	state.Debug = fuse.VerboseTest()
	conn.Debug = fuse.VerboseTest()
	go state.Loop()

	return wd, func() {
		state.Unmount()
		os.RemoveAll(wd)
	}
}

func TestDebug(t *testing.T) {
	wd, clean := setup(t)
	defer clean()

	c, err := ioutil.ReadFile(wd + "/mnt/status/debug")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(c) == 0 {
		t.Fatal("No debug found.")
	}
	log.Println("Found version:", string(c))
}

func TestVersion(t *testing.T) {
	wd, clean := setup(t)
	defer clean()

	c, err := ioutil.ReadFile(wd + "/mnt/status/gounionfs_version")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(c) == 0 {
		t.Fatal("No version found.")
	}
	log.Println("Found version:", string(c))
}

func TestAutoFsSymlink(t *testing.T) {
	wd, clean := setup(t)
	defer clean()

	err := os.Mkdir(wd+"/store/backing1", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	err = os.Symlink(wd+"/ro", wd+"/store/backing1/READONLY")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	err = os.Symlink(wd+"/store/backing1", wd+"/mnt/config/manual1")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	fi, err := os.Lstat(wd + "/mnt/manual1/file1")
	if err != nil {
		t.Fatalf("Lstat failed: %v", err)
	}

	entries, err := ioutil.ReadDir(wd + "/mnt")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 3 {
		t.Error("readdir mismatch", entries)
	}

	err = os.Remove(wd + "/mnt/config/manual1")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	scan := wd + "/mnt/config/" + _SCAN_CONFIG
	err = ioutil.WriteFile(scan, []byte("something"), 0644)
	if err != nil {
		t.Error("error writing:", err)
	}

	fi, _ = os.Lstat(wd + "/mnt/manual1")
	if fi != nil {
		t.Error("Should not have file:", fi)
	}

	_, err = ioutil.ReadDir(wd + "/mnt/config")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	_, err = os.Lstat(wd + "/mnt/backing1/file1")
	if err != nil {
		t.Fatalf("Lstat failed: %v", err)
	}
}

func TestDetectSymlinkedDirectories(t *testing.T) {
	wd, clean := setup(t)
	defer clean()

	err := os.Mkdir(wd+"/backing1", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	err = os.Symlink(wd+"/ro", wd+"/backing1/READONLY")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	err = os.Symlink(wd+"/backing1", wd+"/store/backing1")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	scan := wd + "/mnt/config/" + _SCAN_CONFIG
	err = ioutil.WriteFile(scan, []byte("something"), 0644)
	if err != nil {
		t.Error("error writing:", err)
	}

	_, err = os.Lstat(wd + "/mnt/backing1")
	if err != nil {
		t.Fatalf("Lstat failed: %v", err)
	}
}

func TestExplicitScan(t *testing.T) {
	wd, clean := setup(t)
	defer clean()

	err := os.Mkdir(wd+"/store/backing1", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	os.Symlink(wd+"/ro", wd+"/store/backing1/READONLY")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	fi, _ := os.Lstat(wd + "/mnt/backing1")
	if fi != nil {
		t.Error("Should not have file:", fi)
	}

	scan := wd + "/mnt/config/" + _SCAN_CONFIG
	_, err = os.Lstat(scan)
	if err != nil {
		t.Error(".scan_config missing:", err)
	}

	err = ioutil.WriteFile(scan, []byte("something"), 0644)
	if err != nil {
		t.Error("error writing:", err)
	}

	_, err = os.Lstat(wd + "/mnt/backing1")
	if err != nil {
		t.Error("Should have workspace backing1:", err)
	}
}

func TestCreationChecks(t *testing.T) {
	wd, clean := setup(t)
	defer clean()

	err := os.Mkdir(wd+"/store/foo", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	os.Symlink(wd+"/ro", wd+"/store/foo/READONLY")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	err = os.Mkdir(wd+"/store/ws2", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	os.Symlink(wd+"/ro", wd+"/store/ws2/READONLY")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	err = os.Symlink(wd+"/store/foo", wd+"/mnt/config/bar")
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	err = os.Symlink(wd+"/store/foo", wd+"/mnt/config/foo")
	code := fuse.ToStatus(err)
	if code != fuse.EBUSY {
		t.Error("Should return EBUSY", err)
	}

	err = os.Symlink(wd+"/store/ws2", wd+"/mnt/config/config")
	code = fuse.ToStatus(err)
	if code != fuse.EINVAL {
		t.Error("Should return EINVAL", err)
	}
}
