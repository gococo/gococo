package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/flock"
)

type BuildMutex struct {
	flock   *flock.Flock
	timeout time.Duration
	cancel  context.CancelFunc
}

func NewBuildMutex(path string, timeout time.Duration) *BuildMutex {
	return &BuildMutex{
		flock:   flock.New(path),
		timeout: timeout,
	}
}

func (l *BuildMutex) Lock() (err error) {
	context, cancel := context.WithTimeout(context.Background(), l.timeout)
	l.cancel = cancel

	locked, err := l.flock.TryLockContext(context, time.Second)
	if err != nil {
		return
	}

	if !locked {
		return fmt.Errorf("fail to lock the build")
	}

	return
}

func (l *BuildMutex) Unlock() (err error) {
	defer l.cancel()

	return l.flock.Unlock()
}
