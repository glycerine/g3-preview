package portcheck

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CheckPackageComments rejects provenance and other boilerplate that Go would
// mistake for package documentation. Build constraints remain before the
// package clause; source-origin notes belong after it.
func CheckPackageComments(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".cache" || entry.Name() == ".git") {
				return filepath.SkipDir
			}
			if path != root {
				info, err := os.Stat(filepath.Join(path, "go.mod"))
				if err == nil && !info.IsDir() {
					return filepath.SkipDir
				}
				if err != nil && !os.IsNotExist(err) {
					return err
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		if file.Doc == nil {
			return nil
		}
		want := "Package " + file.Name.Name
		doc := file.Doc.Text()
		if doc == want || strings.HasPrefix(doc, want+" ") {
			return nil
		}
		return fmt.Errorf("%s: pre-package comment is not package documentation; move provenance after the package clause", set.Position(file.Doc.Pos()))
	})
}
