// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

// Package lvdl defines the machine specification types and the
// decoder that reads .lvdl files (s-expression format) into a
// Spec value describing the machine, ISA encoding, and
// instruction set with micro-ops.
package lvdl

import "lvdl-vm/internal/sexpr"

type Spec struct {
	Machine Machine
	ISA     ISA
}

type Machine struct {
	WordBits int
	Regs     []Reg
	Stacks   []Stack
	RAMs     []RAM
}

type Reg struct {
	Name string
	Bits int
}

type Stack struct {
	Name  string
	Words int
	Bits  int
}

type RAM struct {
	Name  string
	Words int
	Bits  int
}

type ISA struct {
	Encoding Encoding
	Instrs   []Instr
	ByName   map[string]Instr
	ByOpcode map[uint8]Instr
}

type Encoding struct {
	OpcodeBits  int
	OperandBits int
}

type Instr struct {
	Name     string
	Opcode   uint8
	Operands []string
	RawMicro []sexpr.Node
	MicroOps []string
}
