package main

import (
	"flag"
	"fmt"
	"github.com/jasonmoo/go-fuse/fuse"
	"github.com/jasonmoo/go-fuse/unionfs"
	"log"
	"os"
	"time"
)

func main() {
	debug := flag.Bool("debug", false, "debug on")
	portable := flag.Bool("portable", false, "use 32 bit inodes")

	entry_ttl := flag.Float64("entry_ttl", 1.0, "fuse entry cache TTL.")
	negative_ttl := flag.Float64("negative_ttl", 1.0, "fuse negative entry cache TTL.")

	delcache_ttl := flag.Float64("deletion_cache_ttl", 5.0, "Deletion cache TTL in seconds.")
	branchcache_ttl := flag.Float64("branchcache_ttl", 5.0, "Branch cache TTL in seconds.")
	deldirname := flag.String(
		"deletion_dirname", "GOUNIONFS_DELETIONS", "Directory name to use for deletions.")

	flag.Parse()
	if len(flag.Args()) < 2 {
		fmt.Println("Usage:\n  unionfs MOUNTPOINT RW-DIRECTORY RO-DIRECTORY ...")
		os.Exit(2)
	}

	ufsOptions := unionfs.UnionFsOptions{
		DeletionCacheTTL: time.Duration(*delcache_ttl * float64(time.Second)),
		BranchCacheTTL:   time.Duration(*branchcache_ttl * float64(time.Second)),
		DeletionDirName:  *deldirname,
	}

	ufs, err := unionfs.NewUnionFsFromRoots(flag.Args()[1:], &ufsOptions, true)
	if err != nil {
		log.Fatal("Cannot create UnionFs", err)
		os.Exit(1)
	}
	nodeFs := fuse.NewPathNodeFs(ufs, &fuse.PathNodeFsOptions{ClientInodes: true})
	mOpts := fuse.FileSystemOptions{
		EntryTimeout:    time.Duration(*entry_ttl * float64(time.Second)),
		AttrTimeout:     time.Duration(*entry_ttl * float64(time.Second)),
		NegativeTimeout: time.Duration(*negative_ttl * float64(time.Second)),
		PortableInodes:  *portable,
	}
	mountState, _, err := fuse.MountNodeFileSystem(flag.Arg(0), nodeFs, &mOpts)
	if err != nil {
		log.Fatal("Mount fail:", err)
	}

	mountState.Debug = *debug
	mountState.Loop()
}
