package build

import (
	"flag"
	"path/filepath"

	"github.com/gococo/gococo/pkg/log"
)

func (b *Build) parseArgs() {
	goFlagSets := flag.NewFlagSet("GO jiayi shi tiancai !!!", flag.ContinueOnError)
	addBuildFlags(goFlagSets)
	addOutputFlags(goFlagSets)
	err := goFlagSets.Parse(b.OriArgs)
	if err != nil {
		log.Fatalf("%v", err)
	}

	flags := make([]string, 0)
	goFlagSets.Visit(func(f *flag.Flag) {
		// 将用户指定 -o 改成绝对目录
		if f.Name == "o" {
			outputDir := f.Value.String()
			outputDir, err := filepath.Abs(outputDir)
			if err != nil {
				log.Fatalf("output flag is not valid: %v", err)
			}
			flags = append(flags, "-o", outputDir)
		} else {
			flags = append(flags, "-"+f.Name, f.Value.String())
		}
	})

	b.NewFlags = flags
	b.NewArgs = goFlagSets.Args()
}

type goConfig struct {
	BuildA                 bool
	BuildBuildmode         string // -buildmode flag
	BuildMod               string // -mod flag
	BuildModReason         string // reason -mod flag is set, if set by default
	BuildI                 bool   // -i flag
	BuildLinkshared        bool   // -linkshared flag
	BuildMSan              bool   // -msan flag
	BuildN                 bool   // -n flag
	BuildO                 string // -o flag
	BuildP                 int    // -p flag
	BuildPkgdir            string // -pkgdir flag
	BuildRace              bool   // -race flag
	BuildToolexec          string // -toolexec flag
	BuildToolchainName     string
	BuildToolchainCompiler func() string
	BuildToolchainLinker   func() string
	BuildTrimpath          bool // -trimpath flag
	BuildV                 bool // -v flag
	BuildWork              bool // -work flag
	BuildX                 bool // -x flag
	// from buildcontext
	Installsuffix string // -installSuffix
	BuildTags     string // -tags
	// from load
	BuildAsmflags   string
	BuildCompiler   string
	BuildGcflags    string
	BuildGccgoflags string
	BuildLdflags    string

	// mod related
	ModCacheRW bool
	ModFile    string
}

var goflags goConfig

func addBuildFlags(cmdSet *flag.FlagSet) {
	cmdSet.BoolVar(&goflags.BuildA, "a", false, "")
	cmdSet.BoolVar(&goflags.BuildN, "n", false, "")
	cmdSet.IntVar(&goflags.BuildP, "p", 4, "")
	cmdSet.BoolVar(&goflags.BuildV, "v", false, "")
	cmdSet.BoolVar(&goflags.BuildX, "x", false, "")
	cmdSet.StringVar(&goflags.BuildBuildmode, "buildmode", "default", "")
	cmdSet.StringVar(&goflags.BuildMod, "mod", "", "")
	cmdSet.StringVar(&goflags.Installsuffix, "installsuffix", "", "")

	// 类型和 go 原生的不一样，这里纯粹是为了 parse 并传递给 go
	cmdSet.StringVar(&goflags.BuildAsmflags, "asmflags", "", "")
	cmdSet.StringVar(&goflags.BuildCompiler, "compiler", "", "")
	cmdSet.StringVar(&goflags.BuildGcflags, "gcflags", "", "")
	cmdSet.StringVar(&goflags.BuildGccgoflags, "gccgoflags", "", "")
	// mod related
	cmdSet.BoolVar(&goflags.ModCacheRW, "modcacherw", false, "")
	cmdSet.StringVar(&goflags.ModFile, "modfile", "", "")
	cmdSet.StringVar(&goflags.BuildLdflags, "ldflags", "", "")
	cmdSet.BoolVar(&goflags.BuildLinkshared, "linkshared", false, "")
	cmdSet.StringVar(&goflags.BuildPkgdir, "pkgdir", "", "")
	cmdSet.BoolVar(&goflags.BuildRace, "race", false, "")
	cmdSet.BoolVar(&goflags.BuildMSan, "msan", false, "")
	cmdSet.StringVar(&goflags.BuildTags, "tags", "", "")
	cmdSet.StringVar(&goflags.BuildToolexec, "toolexec", "", "")
	cmdSet.BoolVar(&goflags.BuildTrimpath, "trimpath", false, "")
	cmdSet.BoolVar(&goflags.BuildWork, "work", false, "")
}

func addOutputFlags(cmdSet *flag.FlagSet) {
	cmdSet.StringVar(&goflags.BuildO, "o", "", "")
}
