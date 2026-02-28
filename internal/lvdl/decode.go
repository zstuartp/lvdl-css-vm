// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package lvdl

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"lvdl-vm/internal/sexpr"
)

var validName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// LoadFile reads a .lvdl spec file and decodes it into a Spec.
func LoadFile(path string) (*Spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	nodes, err := sexpr.ParseAll(strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	return Decode(nodes)
}

// Decode transforms parsed s-expression nodes into a Spec.
// It expects exactly one top-level (lvdl ...) form containing
// (machine ...) and (isa ...) sections. After decoding, ByName
// and ByOpcode lookup maps are populated on the ISA.
func Decode(nodes []sexpr.Node) (*Spec, error) {
	if len(nodes) != 1 {
		return nil, fmt.Errorf(
			"expected exactly 1 top-level form, got %d",
			len(nodes),
		)
	}
	top, ok := nodes[0].(sexpr.List)
	if !ok {
		return nil, fmt.Errorf("top-level must be a list")
	}
	if head(top) != "lvdl" {
		return nil, fmt.Errorf("top-level must be (lvdl ...)")
	}

	var out Spec
	for _, it := range top.Items[1:] {
		sec, ok := it.(sexpr.List)
		if !ok {
			return nil, fmt.Errorf("lvdl child must be list")
		}
		switch head(sec) {
		case "machine":
			m, err := decodeMachine(sec)
			if err != nil {
				return nil, err
			}
			out.Machine = *m
		case "isa":
			isa, err := decodeISA(sec)
			if err != nil {
				return nil, err
			}
			out.ISA = *isa
		default:
			return nil, fmt.Errorf("unknown section %q", head(sec))
		}
	}
	if out.Machine.WordBits == 0 {
		return nil, fmt.Errorf("machine.word is required")
	}

	out.ISA.ByName = map[string]Instr{}
	out.ISA.ByOpcode = map[uint8]Instr{}
	for _, ins := range out.ISA.Instrs {
		up := strings.ToUpper(ins.Name)
		if _, dup := out.ISA.ByName[up]; dup {
			return nil, fmt.Errorf(
				"duplicate instruction name %q", up,
			)
		}
		if _, dup := out.ISA.ByOpcode[ins.Opcode]; dup {
			return nil, fmt.Errorf(
				"duplicate opcode 0x%02X", ins.Opcode,
			)
		}
		out.ISA.ByName[up] = ins
		out.ISA.ByOpcode[ins.Opcode] = ins
	}
	return &out, nil
}

func decodeMachine(lst sexpr.List) (*Machine, error) {
	var m Machine
	for _, it := range lst.Items[1:] {
		form, ok := it.(sexpr.List)
		if !ok {
			return nil, fmt.Errorf("machine entry must be list")
		}
		switch head(form) {
		case "word":
			if len(form.Items) != 2 {
				return nil, fmt.Errorf("(word <bits>)")
			}
			bits, err := mustInt(form.Items[1])
			if err != nil {
				return nil, err
			}
			m.WordBits = bits
		case "reg":
			if len(form.Items) != 3 {
				return nil, fmt.Errorf("(reg NAME bits)")
			}
			name, err := mustSym(form.Items[1])
			if err != nil {
				return nil, err
			}
			bits, err := mustInt(form.Items[2])
			if err != nil {
				return nil, err
			}
			m.Regs = append(m.Regs, Reg{
				Name: name,
				Bits: bits,
			})
		case "stack":
			if len(form.Items) != 4 {
				return nil, fmt.Errorf("(stack NAME words bits)")
			}
			name, err := mustSym(form.Items[1])
			if err != nil {
				return nil, err
			}
			words, err := mustInt(form.Items[2])
			if err != nil {
				return nil, err
			}
			bits, err := mustInt(form.Items[3])
			if err != nil {
				return nil, err
			}
			m.Stacks = append(m.Stacks, Stack{
				Name:  name,
				Words: words,
				Bits:  bits,
			})
		case "ram":
			if len(form.Items) != 4 {
				return nil, fmt.Errorf("(ram NAME words bits)")
			}
			name, err := mustSym(form.Items[1])
			if err != nil {
				return nil, err
			}
			words, err := mustInt(form.Items[2])
			if err != nil {
				return nil, err
			}
			bits, err := mustInt(form.Items[3])
			if err != nil {
				return nil, err
			}
			m.RAMs = append(m.RAMs, RAM{
				Name:  name,
				Words: words,
				Bits:  bits,
			})
		default:
			return nil, fmt.Errorf(
				"unknown machine entry: %q", head(form),
			)
		}
	}
	return &m, nil
}

func decodeISA(lst sexpr.List) (*ISA, error) {
	var isa ISA
	for _, it := range lst.Items[1:] {
		form, ok := it.(sexpr.List)
		if !ok {
			return nil, fmt.Errorf("isa entry must be list")
		}
		switch head(form) {
		case "encoding":
			enc, err := decodeEncoding(form)
			if err != nil {
				return nil, err
			}
			isa.Encoding = *enc
		case "instr":
			ins, err := decodeInstr(form)
			if err != nil {
				return nil, err
			}
			isa.Instrs = append(isa.Instrs, *ins)
		default:
			return nil, fmt.Errorf(
				"unknown isa entry: %q", head(form),
			)
		}
	}
	return &isa, nil
}

func decodeEncoding(lst sexpr.List) (*Encoding, error) {
	var e Encoding
	for _, it := range lst.Items[1:] {
		form, ok := it.(sexpr.List)
		if !ok || len(form.Items) != 2 {
			return nil, fmt.Errorf(
				"encoding entries must be (opcode-bits N) etc",
			)
		}
		switch head(form) {
		case "opcode-bits":
			n, err := mustInt(form.Items[1])
			if err != nil {
				return nil, err
			}
			e.OpcodeBits = n
		case "operand-bits":
			n, err := mustInt(form.Items[1])
			if err != nil {
				return nil, err
			}
			e.OperandBits = n
		default:
			return nil, fmt.Errorf(
				"unknown encoding field: %q", head(form),
			)
		}
	}
	return &e, nil
}

func decodeInstr(lst sexpr.List) (*Instr, error) {
	if len(lst.Items) != 5 {
		return nil, fmt.Errorf(
			"(instr NAME OPCODE (operands...) (micro-ops...))",
		)
	}
	name, err := mustSym(lst.Items[1])
	if err != nil {
		return nil, err
	}
	if !validName.MatchString(name) {
		return nil, fmt.Errorf(
			"invalid instruction name %q: "+
				"must match [A-Za-z_][A-Za-z0-9_]*",
			name,
		)
	}
	op, err := mustUint(lst.Items[2])
	if err != nil {
		return nil, err
	}
	if op > 0xFF {
		return nil, fmt.Errorf("opcode too large: %d", op)
	}

	opsList, ok := lst.Items[3].(sexpr.List)
	if !ok {
		return nil, fmt.Errorf("operands must be a list")
	}
	microList, ok := lst.Items[4].(sexpr.List)
	if !ok {
		return nil, fmt.Errorf("micro-ops must be a list")
	}

	var operands []string
	for _, it := range opsList.Items {
		s, err := mustSym(it)
		if err != nil {
			return nil, fmt.Errorf("operand: %w", err)
		}
		operands = append(operands, s)
	}

	var microOps []string
	var raw []sexpr.Node
	for _, it := range microList.Items {
		raw = append(raw, it)
		microOps = append(microOps, sexpr.Format(it))
	}

	return &Instr{
		Name:     name,
		Opcode:   uint8(op),
		Operands: operands,
		RawMicro: raw,
		MicroOps: microOps,
	}, nil
}

func head(lst sexpr.List) string {
	if len(lst.Items) == 0 {
		return ""
	}
	if a, ok := lst.Items[0].(sexpr.Atom); ok {
		return a.Value
	}
	return ""
}

func mustSym(n sexpr.Node) (string, error) {
	a, ok := n.(sexpr.Atom)
	if !ok {
		return "", fmt.Errorf("expected symbol, got list")
	}
	return a.Value, nil
}

func mustInt(n sexpr.Node) (int, error) {
	u, err := mustUint(n)
	if err != nil {
		return 0, err
	}
	if u > uint64(^uint(0)>>1) {
		return 0, fmt.Errorf("int overflow: %d", u)
	}
	return int(u), nil
}

func mustUint(n sexpr.Node) (uint64, error) {
	a, ok := n.(sexpr.Atom)
	if !ok {
		return 0, fmt.Errorf("expected atom number, got %T", n)
	}
	s := a.Value
	base := 10
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		base = 16
		s = s[2:]
	}
	v, err := strconv.ParseUint(s, base, 64)
	if err != nil {
		return 0, fmt.Errorf("bad number %q: %w", a.Value, err)
	}
	return v, nil
}
