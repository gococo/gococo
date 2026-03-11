package instrument

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Package holds metadata from `go list -json`.
type Package struct {
	Dir        string   `json:"Dir"`
	ImportPath string   `json:"ImportPath"`
	Name       string   `json:"Name"`
	Root       string   `json:",omitempty"`
	GoFiles    []string `json:"GoFiles,omitempty"`
	CgoFiles   []string `json:"CgoFiles,omitempty"`
	Deps       []string `json:"Deps,omitempty"`

	Module   *Module       `json:",omitempty"`
	Goroot   bool          `json:"Goroot,omitempty"`
	Standard bool          `json:"Standard,omitempty"`
	DepOnly  bool          `json:"DepOnly,omitempty"`
	Error    *PackageError `json:"Error,omitempty"`
}

// Module holds go module info.
type Module struct {
	Path      string  `json:",omitempty"`
	Dir       string  `json:",omitempty"`
	GoMod     string  `json:",omitempty"`
	GoVersion string  `json:",omitempty"`
	Main      bool    `json:",omitempty"`
	Replace   *Module `json:",omitempty"`
}

// PackageError holds error info from go list.
type PackageError struct {
	Err string
}

// ListPackages runs `go list -json` and returns all packages keyed by import path.
func ListPackages(dir string, patterns []string) (map[string]*Package, error) {
	args := append([]string{"list", "-json", "-deps"}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("go list failed: %s\n%s", ee.Error(), string(ee.Stderr))
		}
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	pkgs := make(map[string]*Package)
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var p Package
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		pkgs[p.ImportPath] = &p
	}
	return pkgs, nil
}

// FindMainPackages returns packages with Name == "main".
func FindMainPackages(pkgs map[string]*Package) []*Package {
	var mains []*Package
	for _, p := range pkgs {
		if p.Name == "main" {
			mains = append(mains, p)
		}
	}
	return mains
}

// IsProjectPackage returns true if the package belongs to the project (not stdlib, not third-party).
func IsProjectPackage(p *Package, projectModule string) bool {
	if p.Standard || p.Goroot {
		return false
	}
	if p.DepOnly && p.Module != nil && p.Module.Path != projectModule {
		return false
	}
	return strings.HasPrefix(p.ImportPath, projectModule)
}
