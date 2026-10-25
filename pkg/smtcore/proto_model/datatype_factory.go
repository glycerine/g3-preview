package protomodel

import (
	"errors"
	"github.com/glycerine/g3/pkg/ast"
)

type DatatypeFactory struct {
	structuredFactory
	last    map[ast.ASTHandle]ast.ASTHandle
	utility *ast.DatatypeUtil
}

func NewDatatypeFactory(md *ProtoModel) (*DatatypeFactory, error) {
	if err := md.open(); err != nil {
		return nil, err
	}
	fid, err := md.manager.DatatypeFamilyID()
	if err != nil {
		return nil, err
	}
	utility, err := ast.NewDatatypeUtil(md.manager)
	if err != nil {
		return nil, err
	}
	return &DatatypeFactory{structuredFactory: newStructuredFactory(md.manager, fid, md), last: make(map[ast.ASTHandle]ast.ASTHandle), utility: utility}, nil
}
func (f *DatatypeFactory) constructorArguments(c ast.ASTHandle, fresh func(ast.ASTHandle) (ast.ASTHandle, error)) ([]ast.ASTHandle, bool, error) {
	n, err := f.manager.GetArity(c)
	if err != nil {
		return nil, false, err
	}
	args := make([]ast.ASTHandle, n)
	used := false
	for i := range args {
		s, err := f.manager.GetDomain(c, uint32(i))
		if err != nil {
			return nil, false, err
		}
		if !used && fresh != nil {
			args[i], err = fresh(s)
			if err != nil {
				return nil, false, err
			}
			if !args[i].IsNull() {
				used = true
				continue
			}
		}
		args[i], err = f.model.SomeValue(s)
		if err != nil {
			return nil, false, err
		}
	}
	return args, used, nil
}
func (f *DatatypeFactory) SomeValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if len(set.values) != 0 {
		return set.values[0], nil
	}
	c, found, err := f.utility.NonRecursiveConstructor(s)
	if err != nil || !found {
		return ast.ASTHandle{}, err
	}
	args, _, err := f.constructorArguments(c, nil)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	v, err := f.manager.MkApp(c, args)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return v, f.RegisterValue(v)
}
func (f *DatatypeFactory) lastValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	if v := f.last[s]; !v.IsNull() {
		return v, nil
	}
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	var v ast.ASTHandle
	if len(set.values) == 0 {
		v, err = f.SomeValue(s)
	} else {
		v = set.values[0]
	}
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if recursive, e := f.manager.DatatypeIsRecursive(s); e != nil {
		return ast.ASTHandle{}, e
	} else if recursive {
		f.last[s] = v
	}
	return v, nil
}
func (f *DatatypeFactory) contains(root, needle ast.ASTHandle) (bool, error) {
	stack := []ast.ASTHandle{root}
	seen := make(map[ast.ASTHandle]struct{})
	for len(stack) != 0 {
		h := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if h == needle {
			return true, nil
		}
		if _, found := seen[h]; found {
			continue
		}
		seen[h] = struct{}{}
		kind, err := f.manager.GetKind(h)
		if err != nil {
			return false, err
		}
		if kind != ast.ASTApp {
			continue
		}
		n, err := f.manager.GetNumArgs(h)
		if err != nil {
			return false, err
		}
		for i := uint32(0); i < n; i++ {
			a, err := f.manager.GetArg(h, i)
			if err != nil {
				return false, err
			}
			stack = append(stack, a)
		}
	}
	return false, nil
}
func (f *DatatypeFactory) sibling(s, arg ast.ASTHandle) (bool, error) {
	fid, err := f.manager.GetFamilyID(arg)
	if err != nil || fid != f.family {
		return false, err
	}
	return f.manager.DatatypeAreSiblings(s, arg)
}
func (f *DatatypeFactory) almostFresh(s ast.ASTHandle) (ast.ASTHandle, error) {
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if len(set.values) == 0 {
		return f.SomeValue(s)
	}
	constructors, err := f.manager.DatatypeConstructors(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	for _, c := range constructors {
		n, err := f.manager.GetArity(c)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		args := make([]ast.ASTHandle, n)
		fresh, recursive := false, false
		for i := range args {
			a, err := f.manager.GetDomain(c, uint32(i))
			if err != nil {
				return ast.ASTHandle{}, err
			}
			sibling, err := f.sibling(s, a)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			if !fresh && !sibling {
				args[i], err = f.model.FreshValue(a)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				if !args[i].IsNull() {
					fresh = true
					continue
				}
			}
			if !fresh && sibling {
				recursive = true
				args[i], err = f.lastValue(a)
			} else {
				args[i], err = f.model.SomeValue(a)
			}
			if err != nil {
				return ast.ASTHandle{}, err
			}
		}
		if recursive || fresh {
			v, err := f.manager.MkApp(c, args)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			if err = f.RegisterValue(v); err != nil {
				return ast.ASTHandle{}, err
			}
			rec, err := f.manager.DatatypeIsRecursive(s)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			if rec {
				contains, err := f.contains(f.last[s], v)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				if contains {
					return f.last[s], nil
				}
				f.last[s] = v
			}
			return v, nil
		}
	}
	return ast.ASTHandle{}, nil
}
func (f *DatatypeFactory) FreshValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	recursive, err := f.manager.DatatypeIsRecursive(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if len(set.values) == 0 {
		v, err := f.SomeValue(s)
		if err == nil && recursive {
			f.last[s] = v
		}
		return v, err
	}
	constructors, err := f.manager.DatatypeConstructors(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	for _, c := range constructors {
		args, used, err := f.constructorArguments(c, func(a ast.ASTHandle) (ast.ASTHandle, error) {
			if recursive {
				sibling, e := f.sibling(s, a)
				if e != nil || sibling {
					return ast.ASTHandle{}, e
				}
			}
			return f.model.FreshValue(a)
		})
		if err != nil {
			return ast.ASTHandle{}, err
		}
		v, err := f.manager.MkApp(c, args)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if _, found := set.seen[v]; !found {
			if err = f.RegisterValue(v); err != nil {
				return ast.ASTHandle{}, err
			}
			if recursive {
				f.last[s] = v
			}
			return v, nil
		}
		_ = used
	}
	if !recursive {
		return ast.ASTHandle{}, nil
	}
	for iteration := 1; ; iteration++ {
		for _, c := range constructors {
			n, err := f.manager.GetArity(c)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			args := make([]ast.ASTHandle, n)
			foundSibling := false
			for i := range args {
				a, err := f.manager.GetDomain(c, uint32(i))
				if err != nil {
					return ast.ASTHandle{}, err
				}
				sibling, err := f.sibling(s, a)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				if !foundSibling && sibling {
					foundSibling = true
					if iteration == 1 {
						args[i], err = f.almostFresh(a)
					} else {
						args[i], err = f.FreshValue(a)
					}
					if err != nil {
						return ast.ASTHandle{}, err
					}
					if !args[i].IsNull() {
						continue
					}
					foundSibling = false
				}
				args[i], err = f.model.SomeValue(a)
				if err != nil {
					return ast.ASTHandle{}, err
				}
			}
			if foundSibling {
				v, err := f.manager.MkApp(c, args)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				f.last[s] = v
				if _, found := set.seen[v]; !found {
					return v, f.RegisterValue(v)
				}
			}
		}
	}
}
func (f *DatatypeFactory) SomeValues(s ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	return f.structuredFactory.SomeValues(s, f)
}
func (f *DatatypeFactory) Close() error {
	f.last = nil
	var err error
	if f.utility != nil {
		err = f.utility.Close()
	}
	return errors.Join(err, f.structuredFactory.Close())
}
