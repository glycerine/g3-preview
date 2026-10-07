package portcheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// VerifyEvidence checks that target files and named runnable test declarations
// exist. It does not prove tests pass or cover all source behavior; method and
// ownership audits remain explicit and execution gates are separate.
func VerifyEvidence(m Manifest, root string) error {
	// Cache only within this invocation. Every subsequent audit reads current Go
	// declarations, while units/methods sharing a package reuse its parsed names.
	parsedDirs := map[string]map[string]bool{}
	check := func(label string, targets, tests []string) error {
		dirs := map[string]bool{}
		for _, p := range targets {
			if !validPath(p) {
				return fmt.Errorf("%s invalid target %q", label, p)
			}
			info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
			if err != nil {
				return fmt.Errorf("%s target: %w", label, err)
			}
			if info.IsDir() || !strings.HasSuffix(p, ".go") {
				return fmt.Errorf("%s target is not Go source: %s", label, p)
			}
			dirs[filepath.Dir(filepath.Join(root, filepath.FromSlash(p)))] = true
		}
		names := map[string]bool{}
		for dir := range dirs {
			declared, ok := parsedDirs[dir]
			if !ok {
				var err error
				declared, err = evidenceTestDeclarations(dir)
				if err != nil {
					return err
				}
				parsedDirs[dir] = declared
			}
			for name := range declared {
				names[name] = true
			}
		}

		for _, name := range tests {
			if !names[name] {
				return fmt.Errorf("%s missing runnable test declaration %s", label, name)
			}
		}
		return nil
	}
	for _, u := range m.Units {
		if err := check(u.Path, u.GoTargets, u.Tests); err != nil {
			return err
		}
		for _, method := range u.Methods {
			targets := append([]string(nil), method.GoTargets...)
			if method.GoTarget != "" {
				targets = append(targets, method.GoTarget)
			}
			if len(targets) != 0 {
				if err := check(u.Path+"::"+method.Name, targets, method.Tests); err != nil {
					return err
				}
			}
		}
	}
	for _, a := range m.APIs {
		if a.GoTarget != "" {
			if err := check(a.Name, []string{a.GoTarget}, a.Tests); err != nil {
				return err
			}
		}
	}
	return nil
}

// Go toolchain src/cmd/go/internal/load/test.go uses the first rune after the
// entry prefix to distinguish executable tests from ordinary functions.
func evidenceEntryName(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	suffix := name[len(prefix):]
	if suffix == "" {
		return true
	}
	first, _ := utf8.DecodeRuneInString(suffix)
	return !unicode.IsLower(first)
}

func evidenceTestDeclarations(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			return nil, err
		}
		testingAliases := map[string]bool{}
		for _, imp := range file.Imports {
			if imp.Path.Value == `"testing"` {
				name := "testing"
				if imp.Name != nil {
					name = imp.Name.Name
				}
				testingAliases[name] = true
			}
		}
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok || f.Recv != nil || f.Type.Results != nil || f.Type.Params == nil || len(f.Type.Params.List) != 1 {
				continue
			}
			name := f.Name.Name
			kind := ""
			switch {
			case evidenceEntryName(name, "Test"):
				kind = "T"
			case evidenceEntryName(name, "Fuzz"):
				kind = "F"
			case evidenceEntryName(name, "Benchmark"):
				kind = "B"
			default:
				continue
			}
			parameter := f.Type.Params.List[0]
			if len(parameter.Names) > 1 {
				continue
			}
			ptr, ok := parameter.Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			sel, ok := ptr.X.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != kind {
				continue
			}
			pkg, ok := sel.X.(*ast.Ident)
			if ok && testingAliases[pkg.Name] {
				names[name] = true
			}
		}
	}
	return names, nil
}
