package cache

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
	CACHE_PATH   = ".gococo"
	CACHE_DIGEST = "digest.modtime"
)

// BuildCache skip copying if files not changed,
// it also suggests if needs rebuilt
type BuildCache struct {
	oldMod              map[string]int64
	newMod              map[string]int64
	isCacheNeedsRefresh bool

	basePath        string
	cachePath       string
	cacheDigestFile string

	skipPattern map[string]struct{}
}

func NewBuildCache(base string, opts ...Option) (*BuildCache, error) {
	if base == "" {
		return nil, fmt.Errorf("empty base path")
	}

	bc := &BuildCache{
		oldMod:      make(map[string]int64),
		newMod:      make(map[string]int64),
		basePath:    base,
		skipPattern: make(map[string]struct{}),
	}

	if cachePath := os.Getenv("GOCOCO_CACHE_PATH"); cachePath != "" {
		bc.cachePath = filepath.Join(base, cachePath)
	} else {
		bc.cachePath = filepath.Join(base, CACHE_PATH)
	}

	if cacheDigestPath := os.Getenv("GOCOCO_CACHE_DIGEST"); cacheDigestPath != "" {
		bc.cacheDigestFile = filepath.Join(bc.cachePath, cacheDigestPath)
	} else {
		bc.cacheDigestFile = filepath.Join(bc.cachePath, CACHE_DIGEST)
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
	bc.skipPattern[bc.cachePath] = struct{}{}

	return bc, nil
}

// cache the project into the cache folder,
// and calculate the md5 digest of each file
func (bc *BuildCache) Cache() (err error) {
	info, err := os.Lstat(bc.basePath)
	if err != nil {
		return err
	}

	// check mod time
	err = bc.dfs2(bc.basePath, info)
	if err != nil {
		return
	}

	// check if needs to refresh digest
	if !bc.isCacheNeedsRefresh {
		eq := reflect.DeepEqual(bc.newMod, bc.oldMod)
		if eq {
			bc.isCacheNeedsRefresh = false
			log.Donef("files not changed, using old cache")
			return
		} else {
			bc.isCacheNeedsRefresh = true
		}
	}

	// copy project to cache directory
	cache := filepath.Join(bc.cachePath, "cache")
	// remove old tmp if exist
	os.RemoveAll(cache)
	err = os.MkdirAll(cache, os.ModePerm)
	if err != nil {
		return
	}

	// copy recursive
	err = bc.dfs(cache, bc.basePath, info)
	if err != nil {
		return err
	}

	// save the new cache
	return bc.saveCache()
}

// NeedRefresh tells if we need rebuild
func (bc *BuildCache) NeedRefresh() bool {
	return bc.isCacheNeedsRefresh
}

// GetCachePath returns the cache path
func (bc *BuildCache) GetCachePath() string {
	return bc.cachePath
}

func (bc *BuildCache) dfs2(src string, info os.FileInfo) (err error) {

	if _, ok := bc.skipPattern[src]; ok {
		return
	}

	switch {
	case info.Mode()&os.ModeSymlink != 0:
		err = bc.sinfo(src)
	case info.IsDir():
		err = bc.dinfo(src)
	case info.Mode()&os.ModeNamedPipe != 0:
		log.Debugf("skip named pipe file: %v", src)
	default:
		err = bc.finfo(src)
	}

	return
}

func (bc *BuildCache) finfo(src string) (err error) {
	f, err := os.Lstat(src)
	if err != nil {
		return
	}

	bc.newMod[src] = f.ModTime().UnixNano()

	return
}

func (bc *BuildCache) dinfo(src string) (err error) {
	contents, err := os.ReadDir(src)
	if err != nil {
		return
	}

	for _, content := range contents {
		cs := filepath.Join(src, content.Name())

		var err error

		finfo, err := content.Info()
		if err != nil {
			return err
		}

		if err = bc.dfs2(cs, finfo); err != nil {
			return err
		}
	}

	return
}

func (bc *BuildCache) sinfo(src string) (err error) {
	log.Debugf("found symlink: %v, follow the symlink to check mod time", src)
	orig, err := os.Readlink(src)
	if err != nil {
		return
	}

	info, err := os.Lstat(orig)
	if err != nil {
		return
	}

	return bc.dfs2(orig, info)
}

func (bc *BuildCache) dfs(dst string, src string, info os.FileInfo) (err error) {

	if _, ok := bc.skipPattern[src]; ok {
		return
	}

	switch {
	case info.Mode()&os.ModeSymlink != 0:
		err = bc.scopy(dst, src)
	case info.IsDir():
		err = bc.dcopy(dst, src)
	case info.Mode()&os.ModeNamedPipe != 0:
		log.Debugf("skip named pipe file: %v", src)
	default:
		err = bc.fcopy(dst, src)
	}

	return
}

func (bc *BuildCache) fcopy(dst, src string) (err error) {
	if err = os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return
	}

	f, err := os.Create(dst)
	if err != nil {
		return
	}
	defer f.Close()

	s, err := os.Open(src)
	if err != nil {
		return
	}
	defer s.Close()

	if _, err = io.Copy(f, s); err != nil {
		return
	}

	return
}

func (bc *BuildCache) dcopy(dst, src string) (err error) {
	contents, err := os.ReadDir(src)
	if err != nil {
		return
	}

	for _, content := range contents {
		cs := filepath.Join(src, content.Name())
		cd := filepath.Join(dst, content.Name())

		var err error

		finfo, err := content.Info()
		if err != nil {
			return err
		}

		if err = bc.dfs(cd, cs, finfo); err != nil {
			return err
		}
	}

	return
}

// scopy deepcopy the symlink
func (bc *BuildCache) scopy(dst, src string) (err error) {
	log.Debugf("found symlink: %v, deep coping", src)
	orig, err := os.Readlink(src)
	if err != nil {
		return
	}

	info, err := os.Lstat(orig)
	if err != nil {
		return
	}

	return bc.dfs(dst, orig, info)
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
