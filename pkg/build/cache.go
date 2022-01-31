package build

import (
	"bufio"
	"os"
	"strings"

	"github.com/gococo/gococo/pkg/log"
)

const (
	CACHE_DIR = ".gococo"
	CACHE_MD5 = CACHE_DIR + "/digest.md5"
)

type BuildCache struct {
	digest map[string]string
}

func NewBuildCache(path string) *BuildCache {
	digestFile, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open digest file: %s", err)
	}
	defer digestFile.Close()

	scanner := bufio.NewScanner(digestFile)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			log.Fatalf("Invalid digest file: %s", err)
		}
		digest[parts[0]] = parts[1]
	}

	return &BuildCache{
		digest: make(map[string]string),
	}
}
