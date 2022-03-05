package build

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gococo/gococo/pkg/log"
)

func (b *Build) readProjectMetaInfo() {
	b.GOPATH = b.readGOPATH()
	b.GOBIN = b.readGOBIN()

	pkgs := b.listPackages(b.CurWd)

	for _, pkg := range pkgs {
		// check if go mod is enabled
		if pkg.Module == nil {
			log.Fatalf("go module is disabled, gococo only support go mod project")
		}

		b.CurProjectDir = pkg.Module.Dir
		b.ImportPath = pkg.Module.Path

		// no need to loop each package
		break
	}

	// we acutally need package info for the whole project, not only current working directory
	if b.CurWd != b.CurProjectDir {
		b.Pkgs = b.listPackages(b.CurProjectDir)
	} else {
		b.Pkgs = pkgs
	}

	// check if project is built in mod=vendor
	b.IsVendorMod = b.checkIfGoVendor()
	log.Donef("project meta information parsed")
}

func (b *Build) displayProjectMetaInfo() {
	log.Infof("GOBIN: %v", b.GOBIN)
	log.Infof("Project directory: %v", b.CurProjectDir)
	log.Infof("Temporay Project directory: %v", b.CacheProjectDir)

	if b.IsVendorMod {
		log.Infof("mod=vendor")
	}
}

func (b *Build) readGOPATH() string {
	out, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		log.Fatalf("fail to read GOPATH: %v", err)
	}

	return strings.TrimSpace(string(out))
}

func (b *Build) readGOBIN() string {
	out, err := exec.Command("go", "env", "GOBIN").Output()
	if err != nil {
		log.Fatalf("fail to read GOBIN: %v", err)
	}

	return strings.TrimSpace(string(out))
}

// listPackages list all packages under specific via go list command
func (b *Build) listPackages(dir string) map[string]*Package {
	listArgs := []string{"list", "-json"}
	if goflags.BuildTags != "" {
		listArgs = append(listArgs, "-tags", goflags.BuildTags)
	}
	listArgs = append(listArgs, "./...")

	cmd := exec.Command("go", listArgs...)
	cmd.Dir = dir

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("execute go list -json ./... failed, err: %v, stdout: %v, stderr: %v", err, string(out), errBuf.String())
	}

	dec := json.NewDecoder(bytes.NewBuffer(out))
	pkgs := make(map[string]*Package)

	for {
		var pkg Package
		if err := dec.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("reading go list output error: %v", err)
		}
		if pkg.Error != nil {
			log.Fatalf("list package %v failed with error: %v", pkg.ImportPath, pkg.Error)
		}

		pkgs[pkg.ImportPath] = &pkg
	}

	return pkgs
}

func (b *Build) checkIfGoVendor() bool {
	if goflags.BuildMod == "vendor" {
		return true
	}

	vendorDir := filepath.Join(b.CurProjectDir, "vendor")
	if _, err := os.Stat(vendorDir); err != nil {
		return false
	} else {
		return true
	}
}
