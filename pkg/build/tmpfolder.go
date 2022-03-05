package build

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gococo/gococo/pkg/build/sync"
	"github.com/gococo/gococo/pkg/log"
	"golang.org/x/mod/modfile"
)

func (b *Build) copyProjectToTmp() {
	log.StartWait("copying project to temporary directory")

	// get tmp dir for build
	buildCache, err := NewBuildCache(b.CurProjectDir,
		WithPackage(b.Pkgs),
	)

	if err != nil {
		log.Fatalf("fail to initialize the build cache: %v", err)
	}

	// make sure only one gococo is building the project
	buildLock := sync.NewBuildMutex(filepath.Join(b.CurProjectDir, ".gococo.lock"), time.Second*300)
	err = buildLock.Lock()
	if err != nil {
		log.Fatalf("fail to lock the project: %v", err)
	}
	defer buildLock.Unlock()

	// try copy
	err = buildCache.Cache()
	if err != nil {
		log.Fatalf("fail to copy the project: %v", err)
	}

	log.StopWait()

	if buildCache.NeedRefresh() {
		log.Donef("project copied to temporary directory")
	} else {
		log.Donef("project using cache, skip copying to temporary directory")
	}

	b.CacheProjectDir = buildCache.GetCacheDir()
}

// updateGoModfile rewrites the go.mod file in the temporary directory,
//
// if it has a 'replace' directive, and the directive has a relative local path,
// it will be rewritten with a absolute path.
//
// ex.
//
// suppose original project is located at /path/to/aa/bb/cc, go.mod contains a directive:
// 'replace github.com/qiniu/bar =? ../home/foo/bar'
//
// after the project is copied to temporary directory, it should be rewritten as
// 'replace github.com/qiniu/bar => /path/to/aa/bb/home/foo/bar'
func (b *Build) updateGoModFile() {
	tempModfile := filepath.Join(b.CacheProjectDir, "go.mod")
	buf, err := os.ReadFile(tempModfile)
	if err != nil {
		log.Fatalf("cannot read the go.mod file in the temporary directory: %v", err)
	}

	oriGoModFile, err := modfile.Parse(tempModfile, buf, nil)
	if err != nil {
		log.Fatalf("cannot parse the original go.mod: %v", err)
	}

	updateFlag := false
	for index := range oriGoModFile.Replace {
		replace := oriGoModFile.Replace[index]
		oldPath := replace.Old.Path
		oldVersion := replace.Old.Version
		newPath := replace.New.Path
		newVersion := replace.New.Version
		// replace to a local filesystem does not have a version
		// absolute path no need to rewrite
		if newVersion == "" && !filepath.IsAbs(newPath) {
			var absPath string
			fullPath := filepath.Join(b.CacheProjectDir, newPath)
			absPath, _ = filepath.Abs(fullPath)

			_ = oriGoModFile.DropReplace(oldPath, oldVersion)
			_ = oriGoModFile.AddReplace(oldPath, oldVersion, absPath, newVersion)

			updateFlag = true
		}
	}

	oriGoModFile.Cleanup()
	newModFile, _ := oriGoModFile.Format()

	if updateFlag {
		log.Infof("go.mod needs rewrite")
		err := os.WriteFile(tempModfile, newModFile, os.ModePerm)
		if err != nil {
			log.Fatalf("fail to update go.mod: %v", err)
		}
		b.IsModEdit = true
	}
}
