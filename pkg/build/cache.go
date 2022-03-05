package build

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/gococo/gococo/pkg/log"
)

const (
	CACHE_DIR    = ".gococo"
	CACHE_DIGEST = "digest.modtime"
)

// BuildCache skip copying if files not changed,
// it also suggests if needs rebuilt
type BuildCache struct {
	oldMod              map[string]int64
	newMod              map[string]int64
	isCacheNeedsRefresh bool

	baseDir         string
	cacheDir        string
	cacheDigestFile string

	skipPattern map[string]struct{}
	pkgs        []*Package
}

type cacheOption func(*BuildCache)

func WithSkip(p string) cacheOption {
	return func(bc *BuildCache) {
		skipPath := filepath.Join(bc.baseDir, p)
		bc.skipPattern[skipPath] = struct{}{}
	}
}

func WithPackage(pkgs map[string]*Package) cacheOption {
	return func(bc *BuildCache) {
		for _, pkg := range pkgs {
			bc.pkgs = append(bc.pkgs, pkg)
		}
	}
}

func NewBuildCache(base string, opts ...cacheOption) (*BuildCache, error) {
	if base == "" {
		return nil, fmt.Errorf("empty base path")
	}

	bc := &BuildCache{
		oldMod:      make(map[string]int64),
		newMod:      make(map[string]int64),
		baseDir:     base,
		skipPattern: make(map[string]struct{}),
		pkgs:        make([]*Package, 0),
	}

	if cacheDir := os.Getenv("GOCOCO_CACHE_DIR"); cacheDir != "" {
		bc.cacheDir = filepath.Join(base, cacheDir)
	} else {
		bc.cacheDir = filepath.Join(base, CACHE_DIR)
	}

	if cacheDigestPath := os.Getenv("GOCOCO_CACHE_DIGEST"); cacheDigestPath != "" {
		bc.cacheDigestFile = filepath.Join(bc.cacheDir, cacheDigestPath)
	} else {
		bc.cacheDigestFile = filepath.Join(bc.cacheDir, CACHE_DIGEST)
	}

	_, err := os.Lstat(bc.cacheDigestFile)
	if os.IsNotExist(err) {
		// digest file needs update
		bc.isCacheNeedsRefresh = true
	} else if err != nil {
		return nil, err
	} else {
		// load the old digest file
		f, err := os.Open(bc.cacheDigestFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		s := bufio.NewScanner(f)
		for s.Scan() {
			trimed := strings.TrimSpace(s.Text())
			if len(trimed) == 0 {
				continue
			}
			line := strings.Split(trimed, " ")
			if len(line) != 2 {
				return nil, fmt.Errorf("digest file bad format")
			}
			modTime, err := strconv.ParseInt(line[1], 10, 64)
			if err != nil {
				return nil, err
			}
			bc.oldMod[line[0]] = modTime
		}
	}

	// add skip pattern
	for _, o := range opts {
		o(bc)
	}
	// skip cache self
	bc.skipPattern[bc.cacheDir] = struct{}{}

	return bc, nil
}

func (bc *BuildCache) GetCacheDir() string {
	return filepath.Join(bc.cacheDir, "cache")
}

func (bc *BuildCache) Cache() (err error) {

	// get new mod time
	bc.dfsMod()

	// check if needs to refresh digest
	if !bc.isCacheNeedsRefresh {
		eq := reflect.DeepEqual(bc.newMod, bc.oldMod)
		if eq {
			bc.isCacheNeedsRefresh = false
			return
		} else {
			bc.isCacheNeedsRefresh = true
		}
	}

	cacheDir := bc.GetCacheDir()
	// remove old cache
	if err := os.RemoveAll(cacheDir); err != nil {
		return err
	}
	// create new cache dir
	os.MkdirAll(cacheDir, os.ModePerm)

	// copy all files
	bc.dfsCopy()

	// save the new cache
	return bc.saveCache()
}

// NeedRefresh tells if we need rebuild
func (bc *BuildCache) NeedRefresh() bool {
	return bc.isCacheNeedsRefresh
}

func (bc *BuildCache) dfsMod() error {
	srcFiles := make([]string, 0)
	for _, pkg := range bc.pkgs {
		srcFiles = append(srcFiles, bc.sourceFiles(pkg)...)
	}

	for _, src := range srcFiles {
		info, err := os.Lstat(src)
		if err != nil {
			return err
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			log.Debugf("found symlink: %v, follow the symlink to check mod time", src)
			orig, err := os.Readlink(src)
			if err != nil {
				return err
			}

			f, err := os.Stat(orig)
			if err != nil {
				return err
			}

			bc.newMod[src] = f.ModTime().UnixNano()

		default:
			f, err := os.Stat(src)
			if err != nil {
				return err
			}

			bc.newMod[src] = f.ModTime().UnixNano()
		}
	}

	return nil
}

func (bc *BuildCache) dfsCopy() error {
	srcFiles := make([]string, 0)
	modFile := ""
	for _, pkg := range bc.pkgs {
		if modFile == "" {
			modFile = pkg.Module.GoMod
			sumFile := filepath.Join(bc.baseDir, "go.sum")
			srcFiles = append(srcFiles, modFile, sumFile)
		}
		srcFiles = append(srcFiles, bc.sourceFiles(pkg)...)
	}

	projectBase := bc.baseDir

	for _, src := range srcFiles {
		relPath, err := filepath.Rel(projectBase, src)
		if err != nil {
			log.Fatalf("the file is not in the project dir: %v, err: %v", src, err)
		}

		dst := filepath.Join(bc.GetCacheDir(), relPath)
		dstDir := filepath.Dir(dst)
		err = os.MkdirAll(dstDir, os.ModePerm)
		if err != nil {
			log.Fatalf("failed to create dir: %v, %v", dstDir, err)
		}

		f, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer f.Close()

		s, err := os.Open(src)
		if err != nil {
			return err
		}
		defer s.Close()

		if _, err = io.Copy(f, s); err != nil {
			return err
		}
	}

	return nil
}

// save the mod time to disk
func (bc *BuildCache) saveCache() (err error) {
	f, err := os.Create(bc.cacheDigestFile)
	if err != nil {
		return
	}
	defer f.Close()

	for path, modTime := range bc.newMod {
		line := fmt.Sprintf("%v %v\n", path, modTime)
		f.WriteString(line)
	}

	return
}

func (bc *BuildCache) sourceFiles(pkg *Package) []string {
	out := make([]string, 0)
	base := pkg.Dir

	help := func(s string) string {
		return filepath.Join(base, s)
	}

	for _, f := range pkg.GoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.CgoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.CompiledGoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.IgnoredGoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.CFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.IgnoredOtherFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.CXXFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.MFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.HFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.FFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.SFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.SwigFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.SwigCXXFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.SysoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.EmbedFiles {
		out = append(out, help(f))
	}

	return out
}
