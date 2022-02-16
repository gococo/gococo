package build

type Option func(*Build)

func WithBuild() Option {
	return func(b *Build) {
		b.BuildType = GOCOCO_DO_BUILD
	}
}

func WithInstall() Option {
	return func(b *Build) {
		b.BuildType = GOCOCO_DO_INSTALL
	}
}

func WithArgs(args ...string) Option {
	return func(b *Build) {
		b.OriArgs = append(b.OriArgs, args...)
	}
}
