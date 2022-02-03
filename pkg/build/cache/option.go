package cache

import "path/filepath"

type Option func(*BuildCache)

func WithSkip(p string) Option {
	return func(bc *BuildCache) {
		skipPath := filepath.Join(bc.basePath, p)
		bc.skipPattern[skipPath] = struct{}{}
	}
}
