// portcheck is an opt-in maintainer inventory tool; semantic tests are hermetic.
package main

import (
	"flag"
	"fmt"
	"github.com/glycerine/g3/pkg/portcheck"
	"os"
	"path/filepath"
)

func main() {
	root := flag.String("source", "", "reference root; reads reference only when supplied")
	manifest := flag.String("manifest", "../port_manifest.json", "repository ledger path; Go targets are relative to its directory")
	generate := flag.Bool("generate", false, "create initial ledger (refuses overwrite)")
	flag.Parse()
	if err := run(*root, *manifest, *generate); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(root, p string, generate bool) error {
	if generate {
		if root == "" {
			return fmt.Errorf("-generate requires -source")
		}
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("refusing to overwrite %s", p)
		} else if !os.IsNotExist(err) {
			return err
		}
		m, err := portcheck.Inventory(root)
		if err != nil {
			return err
		}
		if err := portcheck.Save(p, m); err != nil {
			return err
		}
		fmt.Printf("inventoried %d source/tool units and %d API operations\n", len(m.Units), len(m.APIs))
		return nil
	}
	evidenceRoot := filepath.Dir(p)
	goRoot := filepath.Join(evidenceRoot, "pkg")
	if info, err := os.Stat(filepath.Join(goRoot, "go.mod")); os.IsNotExist(err) {
		goRoot = evidenceRoot
	} else if err != nil {
		return err
	} else if info.IsDir() {
		return fmt.Errorf("Go module file is a directory: %s", filepath.Join(goRoot, "go.mod"))
	}
	if err := portcheck.CheckNoGenerics(goRoot); err != nil {
		return err
	}
	m, err := portcheck.Load(p)
	if err != nil {
		return err
	}
	if err = portcheck.Validate(m); err != nil {
		return err
	}
	if err := portcheck.VerifyEvidence(m, evidenceRoot); err != nil {
		return err
	}
	if root != "" {
		current, err := portcheck.Inventory(root)
		if err != nil {
			return err
		}
		if err := portcheck.Compare(m, current); err != nil {
			return err
		}
	}
	statuses := map[string]int{}
	for _, u := range m.Units {
		statuses[u.Status]++
	}
	fmt.Printf("%d units, %d APIs: %d ported, %d partial, %d unported, %d dead/deleted\n", len(m.Units), len(m.APIs), statuses["ported"], statuses["partial"], statuses["unported"], statuses["dead_deleted"])
	return nil
}
