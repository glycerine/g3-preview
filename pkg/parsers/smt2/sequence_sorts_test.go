package smt2

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2SequenceSortConstructorsAndSourceRefusals(t *testing.T) {
	for _, tc := range []struct{ name, logic, sort, diagnostic string }{
		{"sequence", "ALL", "(Seq Bool)", ""},
		{"nested", "ALL", "(Seq (Seq Bool))", ""},
		{"regex", "ALL", "(RegEx String)", ""},
		{"regex-nonsequence-source", "ALL", "(RegEx Bool)", ""},
		{"array-index", "ALL", "(Array (Seq Bool) (Seq Bool))", ""},
		{"bare-sequence", "ALL", "Seq", "Invalid sequence sort, expecting one parameter"},
		{"bare-regex", "ALL", "RegEx", "Invalid regex sort, expecting one parameter"},
		{"empty-sequence", "ALL", "(Seq)", "Invalid sequence sort, expecting one parameter"},
		{"two-sequence", "ALL", "(Seq Bool Bool)", "Invalid sequence sort, expecting one parameter"},
		{"empty-regex", "ALL", "(RegEx)", "Invalid regex sort, expecting one parameter"},
		{"two-regex", "ALL", "(RegEx String String)", "Invalid regex sort, expecting one parameter"},
		{"hidden-sequence", "QF_UF", "(Seq Bool)", "unknown sort"},
		{"hidden-regex", "QF_UF", "(RegEx Bool)", "unknown sort"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := c.Close(); err != nil {
					t.Error(err)
				}
			})
			script := "(set-logic " + tc.logic + ")(declare-const value " + tc.sort + ")"
			var output bytes.Buffer
			err = ParseCommands(c, strings.NewReader(script), &output)
			if tc.diagnostic == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
					t.Fatal("sort refusal", err, tc.diagnostic)
				}
				m, err := c.Manager()
				if err != nil {
					t.Fatal(err)
				}
				boolean, err := m.MkBoolSort()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := c.DeclareFunction("value", nil, boolean); err != nil {
					t.Fatal("failed sort installed a declaration", err)
				}
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				command := exec.Command(reference, "-in")
				command.Stdin = strings.NewReader(script)
				native, err := command.CombinedOutput()
				if tc.diagnostic == "" {
					if err != nil || len(native) != 0 {
						t.Fatal("source rejected sort", err, string(native))
					}
				} else if err == nil || !strings.Contains(string(native), tc.diagnostic) {
					t.Fatal("source refusal", err, string(native), tc.diagnostic)
				}
			}
		})
	}
}
