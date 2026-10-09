package portcheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CheckNoGenerics guards project declarations and explicit type-argument syntax
// in production, tests and generated Go files. It also rejects generic standard
// library calls used without explicit type arguments. Ordinary array indexing
// remains valid. Nested Go modules are audited separately at their own roots.
// This syntax audit does not replace review of external APIs.
func CheckNoGenerics(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && (d.Name() == ".cache" || d.Name() == ".git") {
				return filepath.SkipDir
			}
			if p != root {
				info, err := os.Stat(filepath.Join(p, "go.mod"))
				if err == nil && !info.IsDir() {
					return filepath.SkipDir
				}
				if err != nil && !os.IsNotExist(err) {
					return err
				}
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		set := token.NewFileSet()
		f, err := parser.ParseFile(set, p, nil, 0)
		if err != nil {
			return err
		}
		imports := map[string]string{}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			alias := filepath.Base(path)
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			imports[alias] = path
		}
		var problem error
		fail := func(n ast.Node) {
			if problem == nil {
				problem = fmt.Errorf("%s: Go generics prohibited by AGENTS.md", set.Position(n.Pos()))
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if problem != nil {
				return false
			}
			switch x := n.(type) {
			case *ast.FuncType:
				if x.TypeParams != nil && len(x.TypeParams.List) > 0 {
					fail(x)
				}
			case *ast.TypeSpec:
				if x.TypeParams != nil && len(x.TypeParams.List) > 0 {
					fail(x)
				}
			case *ast.UnaryExpr:
				if x.Op == token.TILDE {
					fail(x)
				}
			case *ast.InterfaceType:
				for _, field := range x.Methods.List {
					if term, ok := field.Type.(*ast.BinaryExpr); ok && term.Op == token.OR {
						fail(term)
					}
				}
			case *ast.IndexListExpr:
				if !approvedGenericInstantiation(x.X, imports) {
					fail(x)
				}
			case *ast.IndexExpr:
				if approvedGenericInstantiation(x.X, imports) {
					break
				}
				// A known type in an index position denotes a single type argument.
				if id, ok := x.Index.(*ast.Ident); ok {
					if (id.Obj != nil && id.Obj.Kind == ast.Typ) || builtinType(id.Name) {
						fail(x)
					}
				}
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok {
					switch imports[id.Name] {
					case "slices", "maps", "cmp", "unique", "iter", "weak":
						fail(x)
					case "reflect":
						if x.Sel.Name == "TypeFor" {
							fail(x)
						}
					case "errors":
						if x.Sel.Name == "AsType" {
							fail(x)
						}
					}
				}
			}
			return problem == nil
		})
		return problem
	})
}

func approvedGenericInstantiation(expr ast.Expr, imports map[string]string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	qualifier, ok := selector.X.(*ast.Ident)
	return ok && imports[qualifier.Name] == "github.com/glycerine/insdict"
}

func builtinType(name string) bool {
	switch name {
	case "bool", "string", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune", "float32", "float64", "complex64", "complex128", "any":
		return true
	}
	return false
}
