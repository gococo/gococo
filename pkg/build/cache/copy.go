package cache

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/gococo/gococo/pkg/log"
)

const (
	CACHE_PATH   = ".gococo"
	CACHE_DIGEST = "digest.md5"
)

type BuildCache struct {
	oldDigest   map[string]string
	newDigest   map[string]string
	isNewDigest bool

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
		oldDigest:   make(map[string]string),
		newDigest:   make(map[string]string),
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
		bc.isNewDigest = true
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
			bc.oldDigest[line[0]] = line[1]
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

	// copy project to tmp directory and caclulate md5
	tmp := filepath.Join(bc.cachePath, "tmp")
	// remove old tmp if exist
	os.RemoveAll(tmp)

	// copy recursive
	err = bc.dfs(tmp, bc.basePath, info)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	// check if needs to refresh digest
	if !bc.isNewDigest {
		eq := reflect.DeepEqual(bc.newDigest, bc.oldDigest)
		if eq {
			bc.isNewDigest = false
			log.Donef("files not changed, using old cache")
			return
		} else {
			bc.isNewDigest = true
		}
	}

	// move tmp to cache
	cache := filepath.Join(bc.cachePath, "cache")
	// rm old cache
	os.RemoveAll(cache)
	os.Rename(tmp, cache)

	// save new md5 digest
	return bc.saveDigest()
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

	w := &DigestWriter{
		f:   f,
		md5: md5.New(),
	}
	if _, err = io.Copy(w, s); err != nil {
		return
	}

	relPath, err := filepath.Rel(bc.basePath, src)
	if err != nil {
		return
	}
	bc.newDigest[relPath] = fmt.Sprintf("%x", w.md5.Sum(nil))

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

func (bc *BuildCache) saveDigest() (err error) {
	f, err := os.Create(bc.cacheDigestFile)
	if err != nil {
		return
	}
	defer f.Close()

	for k, v := range bc.newDigest {
		line := k + " " + v + "\n"
		_, err = f.WriteString(line)
		if err != nil {
			return
		}
	}

	return
}
