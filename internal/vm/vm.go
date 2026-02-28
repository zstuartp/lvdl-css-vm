// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

// Package vm implements the LVDL virtual machine interpreter.
// It executes a program (slice of 64-bit words) against an ISA
// defined in an lvdl.Spec, recording a full state trace for
// each step.
package vm

import (
	"fmt"
	"strings"

	"lvdl-vm/internal/lvdl"
)

type State struct {
	Step   int
	PC     uint64
	SP     uint64
	ACC    uint64
	Stack  []uint64
	Output []uint64
	Halted bool
}

type Machine struct {
	PC  uint64
	SP  uint64
	ACC uint64

	Stack  []uint64
	Prog   []uint64
	Output []uint64

	ISA lvdl.ISA
}

// NewMachine creates a machine initialized with the given program
// and stack size from the spec (default 256 words).
func NewMachine(spec *lvdl.Spec, prog []uint64) (*Machine, error) {
	stackWords := 256
	if len(spec.Machine.Stacks) > 0 && spec.Machine.Stacks[0].Words > 0 {
		stackWords = spec.Machine.Stacks[0].Words
	}
	return &Machine{
		PC: 0, SP: 0, ACC: 0,
		Stack: make([]uint64, stackWords),
		Prog:  prog,
		ISA:   spec.ISA,
	}, nil
}

// Run executes up to maxSteps instructions, returning a trace
// of states. trace[0] is the initial state before any execution.
func (m *Machine) Run(maxSteps int) ([]State, error) {
	var trace []State
	halted := false
	trace = append(trace, m.snapshot(0, halted))

	for step := 1; step <= maxSteps; step++ {
		if halted {
			trace = append(trace, m.snapshot(step, halted))
			continue
		}
		if int(m.PC) >= len(m.Prog) {
			return trace, fmt.Errorf(
				"pc out of program: PC=%d len=%d",
				m.PC, len(m.Prog),
			)
		}
		ins := m.Prog[m.PC]
		m.PC++

		op := uint8(ins >> 56)
		imm := ins & 0x00FFFFFFFFFFFFFF

		instr, ok := m.ISA.ByOpcode[op]
		if !ok {
			return trace, fmt.Errorf("unimplemented opcode 0x%02X at step %d", op, step)
		}

		switch strings.ToUpper(instr.Name) {
		case "HLT":
			halted = true
		case "PSH":
			if int(m.SP) >= len(m.Stack) {
				return trace, fmt.Errorf("stack overflow: SP=%d", m.SP)
			}
			m.Stack[m.SP] = imm
			m.SP++
		case "POP":
			if m.SP == 0 {
				return trace, fmt.Errorf("stack underflow: POP at step %d", step)
			}
			m.SP--
		case "DUP":
			if m.SP == 0 {
				return trace, fmt.Errorf("stack underflow: DUP at step %d", step)
			}
			if int(m.SP) >= len(m.Stack) {
				return trace, fmt.Errorf("stack overflow: DUP at step %d", step)
			}
			m.Stack[m.SP] = m.Stack[m.SP-1]
			m.SP++
		case "ADD":
			if m.SP < 2 {
				return trace, fmt.Errorf("stack underflow: ADD at step %d", step)
			}
			b := m.Stack[m.SP-1]
			a := m.Stack[m.SP-2]
			m.SP--
			m.Stack[m.SP-1] = a + b
		case "NEG":
			if m.SP == 0 {
				return trace, fmt.Errorf("stack underflow: NEG at step %d", step)
			}
			m.Stack[m.SP-1] = uint64(-int64(m.Stack[m.SP-1]))
		case "JMP":
			m.PC = imm
		case "JEZ":
			if m.SP == 0 {
				return trace, fmt.Errorf("stack underflow: JEZ at step %d", step)
			}
			m.SP--
			val := m.Stack[m.SP]
			if val == 0 {
				m.PC = imm
			}
		case "CLL":
			if int(m.SP) >= len(m.Stack) {
				return trace, fmt.Errorf("stack overflow: CLL at step %d", step)
			}
			// PC already points at the next instruction after fetch/decode.
			m.Stack[m.SP] = m.PC
			m.SP++
			m.PC = imm
		case "RET":
			if m.SP == 0 {
				return trace, fmt.Errorf("stack underflow: RET at step %d", step)
			}
			m.SP--
			m.PC = m.Stack[m.SP]
		case "SAY":
			if m.SP == 0 {
				return trace, fmt.Errorf("stack underflow: SAY at step %d", step)
			}
			m.Output = append(m.Output, m.Stack[m.SP-1])
		case "SWP":
			if m.SP < 2 {
				return trace, fmt.Errorf("stack underflow: SWP at step %d", step)
			}
			m.Stack[m.SP-1], m.Stack[m.SP-2] = m.Stack[m.SP-2], m.Stack[m.SP-1]
		case "OVR":
			if m.SP < 2 {
				return trace, fmt.Errorf("stack underflow: OVR at step %d", step)
			}
			if int(m.SP) >= len(m.Stack) {
				return trace, fmt.Errorf("stack overflow: OVR at step %d", step)
			}
			m.Stack[m.SP] = m.Stack[m.SP-2]
			m.SP++
		case "ROT":
			if m.SP < 3 {
				return trace, fmt.Errorf("stack underflow: ROT at step %d", step)
			}
			a := m.Stack[m.SP-3]
			b := m.Stack[m.SP-2]
			c := m.Stack[m.SP-1]
			m.Stack[m.SP-3] = b
			m.Stack[m.SP-2] = c
			m.Stack[m.SP-1] = a
		case "PCK":
			depth := imm
			if depth >= m.SP {
				return trace, fmt.Errorf(
					"stack underflow: PCK %d at step %d", imm, step,
				)
			}
			if int(m.SP) >= len(m.Stack) {
				return trace, fmt.Errorf("stack overflow: PCK at step %d", step)
			}
			m.Stack[m.SP] = m.Stack[m.SP-1-depth]
			m.SP++
		case "RLL":
			if m.SP < 3 {
				return trace, fmt.Errorf("stack underflow: RLL at step %d", step)
			}
			a := m.Stack[m.SP-3]
			b := m.Stack[m.SP-2]
			c := m.Stack[m.SP-1]
			m.Stack[m.SP-3] = c
			m.Stack[m.SP-2] = a
			m.Stack[m.SP-1] = b
		default:
			return trace, fmt.Errorf("unimplemented instruction %q (0x%02X) at step %d", instr.Name, op, step)
		}

		trace = append(trace, m.snapshot(step, halted))
	}
	return trace, nil
}

func (m *Machine) snapshot(step int, halted bool) State {
	n := int(m.SP)
	if n > len(m.Stack) {
		n = len(m.Stack)
	}
	cp := make([]uint64, n)
	copy(cp, m.Stack[:n])
	out := make([]uint64, len(m.Output))
	copy(out, m.Output)
	return State{
		Step:   step,
		PC:     m.PC,
		SP:     m.SP,
		ACC:    m.ACC,
		Stack:  cp,
		Output: out,
		Halted: halted,
	}
}
