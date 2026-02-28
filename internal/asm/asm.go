// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

// Package asm provides a small two-pass assembler for the 8-bit CSS runtime
// branch. It encodes 16-bit instruction words as [opcode:8 | operand:8].
package asm

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"lvdl-vm/internal/lvdl"
	"lvdl-vm/internal/profile"
)

type Program struct {
	Words []uint16
}

// Options controls assembler behavior for runtime profile limits.
type Options struct {
	MaxProgramPC int
}

func normalizedOptions(opts Options) Options {
	if opts.MaxProgramPC <= 0 {
		opts.MaxProgramPC = profile.Runtime8.MaxProgramPC()
	}
	return opts
}

// Assemble reads assembly text from r and returns the encoded program.
// Supported mnemonics in this branch are: HLT, PSH, POP, ADD, SUB, AND, XOR, OR, DUP, SWP, OVR, INC, DEC, CLL, RET, NOT, SHL, SHR, JMP, JNZ, SAY.
func Assemble(r io.Reader, isa lvdl.ISA) (*Program, error) {
	return AssembleWithOptions(r, isa, Options{})
}

// AssembleWithOptions reads assembly text from r and returns the encoded program.
func AssembleWithOptions(r io.Reader, isa lvdl.ISA, opts Options) (*Program, error) {
	opts = normalizedOptions(opts)
	sc := bufio.NewScanner(r)

	type pending struct {
		op   string
		arg  string
		line int
	}

	var lines []pending
	labels := map[string]int{}
	pc := 0
	ln := 0

	for sc.Scan() {
		ln++
		s := stripComment(sc.Text())
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		if strings.HasSuffix(s, ":") {
			lab := strings.TrimSpace(strings.TrimSuffix(s, ":"))
			if lab == "" {
				return nil, fmt.Errorf("line %d: empty label", ln)
			}
			if _, exists := labels[lab]; exists {
				return nil, fmt.Errorf("line %d: duplicate label %q", ln, lab)
			}
			labels[lab] = pc
			continue
		}

		parts := strings.Fields(s)
		if len(parts) > 2 {
			return nil, fmt.Errorf("line %d: too many fields in %q", ln, s)
		}

		op := strings.ToUpper(parts[0])
		arg := ""
		if len(parts) == 2 {
			arg = parts[1]
		}

		lines = append(lines, pending{op: op, arg: arg, line: ln})
		pc++
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	var out Program
	for _, p := range lines {
		switch p.op {
		case "HLT", "POP", "ADD", "SUB", "AND", "XOR", "OR", "DUP", "SWP", "OVR", "INC", "DEC", "RET", "NOT", "SHL", "SHR", "SAY":
			if p.arg != "" {
				return nil, fmt.Errorf("line %d: %s takes no operand", p.line, p.op)
			}
			op, err := opcodeFor(isa, p.op)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			out.Words = append(out.Words, encode(op, 0))

		case "PSH":
			if p.arg == "" {
				return nil, fmt.Errorf("line %d: PSH requires imm8", p.line)
			}
			op, err := opcodeFor(isa, "PSH")
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			imm, err := parseImm8(p.arg)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			out.Words = append(out.Words, encode(op, imm))

		case "JMP":
			if p.arg == "" {
				return nil, fmt.Errorf("line %d: JMP requires target", p.line)
			}
			op, err := opcodeFor(isa, "JMP")
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			target, err := parseJumpTarget(p.arg, labels)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			if int(target) > opts.MaxProgramPC {
				return nil, fmt.Errorf(
					"line %d: JMP target out of range (0..%d): %d",
					p.line, opts.MaxProgramPC, target,
				)
			}
			out.Words = append(out.Words, encode(op, target))

		case "JNZ":
			if p.arg == "" {
				return nil, fmt.Errorf("line %d: JNZ requires target", p.line)
			}
			op, err := opcodeFor(isa, "JNZ")
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			target, err := parseJumpTarget(p.arg, labels)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			if int(target) > opts.MaxProgramPC {
				return nil, fmt.Errorf(
					"line %d: JNZ target out of range (0..%d): %d",
					p.line, opts.MaxProgramPC, target,
				)
			}
			out.Words = append(out.Words, encode(op, target))

		case "CLL":
			if p.arg == "" {
				return nil, fmt.Errorf("line %d: CLL requires target", p.line)
			}
			op, err := opcodeFor(isa, "CLL")
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			target, err := parseJumpTarget(p.arg, labels)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", p.line, err)
			}
			if int(target) > opts.MaxProgramPC {
				return nil, fmt.Errorf(
					"line %d: CLL target out of range (0..%d): %d",
					p.line, opts.MaxProgramPC, target,
				)
			}
			out.Words = append(out.Words, encode(op, target))

		default:
			return nil, fmt.Errorf(
				"line %d: assembler supports only HLT/PSH/POP/ADD/SUB/AND/XOR/OR/DUP/SWP/OVR/INC/DEC/CLL/RET/NOT/SHL/SHR/JMP/JNZ/SAY in this branch (got %q)",
				p.line, p.op,
			)
		}
	}

	return &out, nil
}

func opcodeFor(isa lvdl.ISA, name string) (uint8, error) {
	ins, ok := isa.ByName[name]
	if !ok {
		return 0, fmt.Errorf("ISA missing instruction %q", name)
	}
	return ins.Opcode, nil
}

func encode(opcode uint8, operand uint8) uint16 {
	return (uint16(opcode) << 8) | uint16(operand)
}

func stripComment(s string) string {
	if i := strings.IndexByte(s, ';'); i >= 0 {
		return s[:i]
	}
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i]
	}
	return s
}

func parseImm8(tok string) (uint8, error) {
	s := tok
	base := 10
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		base = 16
		s = s[2:]
	} else if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		base = 2
		s = s[2:]
	}
	n, err := strconv.ParseUint(s, base, 8)
	if err != nil {
		return 0, fmt.Errorf("bad imm8 %q", tok)
	}
	return uint8(n), nil
}

func parseJumpTarget(tok string, labels map[string]int) (uint8, error) {
	if imm, err := parseImm8(tok); err == nil {
		return imm, nil
	}
	pc, ok := labels[tok]
	if !ok {
		return 0, fmt.Errorf("bad jump target %q", tok)
	}
	if pc < 0 || pc > 255 {
		return 0, fmt.Errorf("jump target out of imm8 range: %d", pc)
	}
	return uint8(pc), nil
}
