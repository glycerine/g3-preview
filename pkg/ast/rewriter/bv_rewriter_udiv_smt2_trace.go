// Source: src/ast/rewriter/bv_rewriter.cpp:1040,1097; src/util/trace.h:40-46.
package rewriter

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (p *BVRewriter) unsignedDivisionTraceTags() *diagnostic.TraceTags {
	if p.trace != nil {
		return p.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (p *BVRewriter) writeUnsignedDivisionSMT2(w io.Writer, h ast.ASTHandle, globals *uparams.GlobalParamsService) (result error) {
	format, err := p.manager.SMT2PPExprTree(h, nil)
	if err != nil {
		return err
	}
	stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: p.manager.Budget()})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, stream.Close()) }()
	empty := globals.Context.EmptyValue()
	if err = p.manager.PrintFormat(stream, format, globals.Context, &empty, globals); err != nil {
		return err
	}
	body, err := stream.String()
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, body)
	return err
}

func (p *BVRewriter) traceUnsignedDivisionOption(rhs ast.ASTHandle) error {
	tags := p.unsignedDivisionTraceTags()
	if !tags.ShouldTrace("udiv2mul") {
		return nil
	}
	globals := uparams.BorrowProcessGlobalParamsService()
	if globals == nil || globals.Context == nil {
		return nil
	}
	return tags.EmitConditional("udiv2mul", "mk_bv_udiv_core", "src/ast/rewriter/bv_rewriter.cpp", 1040,
		func() bool { return true }, func(w io.Writer) error {
			if err := p.writeUnsignedDivisionSMT2(w, rhs, globals); err != nil {
				return err
			}
			text := "0"
			if p.polynomial.config.bitVector.UnsignedDivisionToMultiply {
				text = "1"
			}
			_, err := io.WriteString(w, " udiv2mul: "+text+"\n")
			return err
		})
}

func (p *BVRewriter) traceUnsignedDivisionRewrite(lhs, rhs, result ast.ASTHandle) error {
	tags := p.unsignedDivisionTraceTags()
	if !tags.ShouldTrace("bv_udiv") {
		return nil
	}
	globals := uparams.BorrowProcessGlobalParamsService()
	if globals == nil || globals.Context == nil {
		return nil
	}
	return tags.EmitConditional("bv_udiv", "mk_bv_udiv_core", "src/ast/rewriter/bv_rewriter.cpp", 1097,
		func() bool { return true }, func(w io.Writer) error {
			for _, h := range [...]ast.ASTHandle{lhs, rhs} {
				if err := p.writeUnsignedDivisionSMT2(w, h, globals); err != nil {
					return err
				}
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, "---->\n"); err != nil {
				return err
			}
			if err := p.writeUnsignedDivisionSMT2(w, result, globals); err != nil {
				return err
			}
			_, err := io.WriteString(w, "\n")
			return err
		})
}
