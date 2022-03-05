package build

import (
	"os"
	"os/exec"
	"strings"

	"github.com/gococo/gococo/pkg/log"
)

const (
	GOCOCO_DO_BUILD = iota
	GOCOCO_DO_INSTALL
)

type Build struct {
	OriArgs []string

	NewFlags  []string
	NewArgs   []string
	BuildType int

	GOPATH      string
	GOBIN       string
	IsModEdit   bool // is the mod file edited in cache dir
	IsVendorMod bool // is the project turn on vendor

	CurWd           string
	CacheWd         string
	CurProjectDir   string
	CacheProjectDir string

	ImportPath string
	Pkgs       map[string]*Package
}

func NewBuild(opts ...Option) *Build {
	b := &Build{
		OriArgs: make([]string, 0),
	}

	for _, o := range opts {
		o(b)
	}

	// get current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot get current working directory: %v", wd)
	}
	b.CurWd = wd

	// parse flags and args,
	b.parseArgs()

	// get project meta information
	b.readProjectMetaInfo()

	// copy to tmp project
	b.copyProjectToTmp()

	// display meta info
	b.displayProjectMetaInfo()

	return b
}

func (b *Build) Build() {
	b.updateGoModFile()

	b.doBuildInTemp()
}

func (b *Build) doBuildInTemp() {
	log.StartWait("building the injected project")

	goflags := b.NewFlags

	if b.IsModEdit && b.IsVendorMod {
		// is there a better solution? as we change the vendor to readonly...
		goflags = append(goflags, "-mod", "readonly")
	}

	// chech if -o is set
	oSet := false
	for _, flag := range goflags {
		if flag == "-o" {
			oSet = true
		}
	}

	// is not set, output the binary to original working directory
	if !oSet {
		goflags = append(goflags, "-o", b.CurWd)
	}

	args := []string{"build"}
	args = append(args, goflags...)
	args = append(args, b.NewArgs...)

	cmd := exec.Command("go", args...)
	cmd.Dir = b.CacheProjectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Infof("go build cmd is: %v, in path [%v]", nicePrintArgs(cmd.Args), cmd.Dir)
	if err := cmd.Start(); err != nil {
		log.Fatalf("fail to execute go build: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatalf("fail to execute go build: %v", err)
	}

	log.StopWait()
	log.Donef("go build done")
}

// ex.
// go build -ldflags "-X mmm" -o /home/lyy/cmd . => go build -ldflags -X mmm -o /home/lyy/cmd .
// so we need nice print
func nicePrintArgs(args []string) []string {
	output := make([]string, 0)
	for _, arg := range args {
		if strings.Contains(arg, " ") {
			output = append(output, "\""+arg+"\"")
		} else {
			output = append(output, arg)
		}
	}

	return output
}
