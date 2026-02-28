// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package vm

import (
	"strings"
	"testing"

	"lvdl-vm/internal/lvdl"
)

func testSpec() *lvdl.Spec {
	instrs := []struct {
		name string
		op   uint8
	}{
		{"HLT", 0x00},
		{"PSH", 0x01},
		{"POP", 0x02},
		{"DUP", 0x03},
		{"ADD", 0x04},
		{"NEG", 0x05},
		{"JMP", 0x06},
		{"JEZ", 0x07},
		{"SAY", 0x08},
		{"SWP", 0x09},
		{"OVR", 0x0A},
		{"ROT", 0x0B},
		{"PCK", 0x0C},
		{"RLL", 0x0D},
		{"CLL", 0x0E},
		{"RET", 0x0F},
	}

	byName := make(map[string]lvdl.Instr, len(instrs))
	byOpcode := make(map[uint8]lvdl.Instr, len(instrs))
	for _, in := range instrs {
		ins := lvdl.Instr{Name: in.name, Opcode: in.op}
		byName[in.name] = ins
		byOpcode[in.op] = ins
	}

	return &lvdl.Spec{
		Machine: lvdl.Machine{
			Stacks: []lvdl.Stack{{Name: "STK", Words: 256, Bits: 64}},
		},
		ISA: lvdl.ISA{
			ByName:   byName,
			ByOpcode: byOpcode,
		},
	}
}

// w encodes a 64-bit instruction word: [opcode:8 | operand:56].
func w(op, imm uint64) uint64 { return (op << 56) | (imm & 0x00FFFFFFFFFFFFFF) }

func TestRunROT(t *testing.T) {
	spec := testSpec()
	prog := []uint64{
		w(0x01, 1), // PSH 1
		w(0x01, 2), // PSH 2
		w(0x01, 3), // PSH 3
		w(0x0B, 0), // ROT
		w(0x00, 0), // HLT
	}

	m, err := NewMachine(spec, prog)
	if err != nil {
		t.Fatalf("NewMachine() error = %v", err)
	}
	trace, err := m.Run(16)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	last := trace[len(trace)-1]
	if last.SP != 3 {
		t.Fatalf("SP = %d, want 3", last.SP)
	}
	want := []uint64{2, 3, 1}
	for i, wv := range want {
		if got := last.Stack[i]; got != wv {
			t.Fatalf("stack[%d] = %d, want %d", i, got, wv)
		}
	}
}

func TestRunPCK(t *testing.T) {
	spec := testSpec()
	prog := []uint64{
		w(0x01, 10), // PSH 10
		w(0x01, 20), // PSH 20
		w(0x01, 30), // PSH 30
		w(0x0C, 2),  // PCK 2
		w(0x00, 0),  // HLT
	}

	m, err := NewMachine(spec, prog)
	if err != nil {
		t.Fatalf("NewMachine() error = %v", err)
	}
	trace, err := m.Run(16)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	last := trace[len(trace)-1]
	if last.SP != 4 {
		t.Fatalf("SP = %d, want 4", last.SP)
	}
	want := []uint64{10, 20, 30, 10}
	for i, wv := range want {
		if got := last.Stack[i]; got != wv {
			t.Fatalf("stack[%d] = %d, want %d", i, got, wv)
		}
	}
}

func TestRunRLL(t *testing.T) {
	spec := testSpec()
	prog := []uint64{
		w(0x01, 1), // PSH 1
		w(0x01, 2), // PSH 2
		w(0x01, 3), // PSH 3
		w(0x0D, 0), // RLL
		w(0x00, 0), // HLT
	}

	m, err := NewMachine(spec, prog)
	if err != nil {
		t.Fatalf("NewMachine() error = %v", err)
	}
	trace, err := m.Run(16)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	last := trace[len(trace)-1]
	if last.SP != 3 {
		t.Fatalf("SP = %d, want 3", last.SP)
	}
	want := []uint64{3, 1, 2}
	for i, wv := range want {
		if got := last.Stack[i]; got != wv {
			t.Fatalf("stack[%d] = %d, want %d", i, got, wv)
		}
	}
}

func TestRunCLLRET(t *testing.T) {
	spec := testSpec()
	prog := []uint64{
		w(0x0E, 3), // 0: CLL 3
		w(0x00, 0), // 1: HLT
		w(0x00, 0), // 2: HLT (padding)
		w(0x01, 7), // 3: PSH 7
		w(0x08, 0), // 4: SAY
		w(0x02, 0), // 5: POP (discard local value)
		w(0x0F, 0), // 6: RET
	}

	m, err := NewMachine(spec, prog)
	if err != nil {
		t.Fatalf("NewMachine() error = %v", err)
	}
	trace, err := m.Run(32)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	last := trace[len(trace)-1]
	if last.PC != 2 {
		t.Fatalf("PC = %d, want 2 (halted after return to caller)", last.PC)
	}
	if last.SP != 0 {
		t.Fatalf("SP = %d, want 0", last.SP)
	}
	if len(last.Output) != 1 || last.Output[0] != 7 {
		t.Fatalf("Output = %v, want [7]", last.Output)
	}
}

func TestRunRETUnderflow(t *testing.T) {
	spec := testSpec()
	prog := []uint64{
		w(0x0F, 0), // RET
	}

	m, err := NewMachine(spec, prog)
	if err != nil {
		t.Fatalf("NewMachine() error = %v", err)
	}
	_, err = m.Run(8)
	if err == nil {
		t.Fatal("Run() error = nil, want underflow error")
	}
	if !strings.Contains(err.Error(), "stack underflow: RET") {
		t.Fatalf("Run() error = %q, want RET underflow", err)
	}
}

func TestRunPCKUnderflow(t *testing.T) {
	spec := testSpec()
	prog := []uint64{
		w(0x01, 1), // PSH 1
		w(0x0C, 1), // PCK 1  (only 1 element on stack, depth 1 invalid)
		w(0x00, 0), // HLT
	}

	m, err := NewMachine(spec, prog)
	if err != nil {
		t.Fatalf("NewMachine() error = %v", err)
	}
	_, err = m.Run(8)
	if err == nil {
		t.Fatal("Run() error = nil, want underflow error")
	}
	if !strings.Contains(err.Error(), "stack underflow: PCK") {
		t.Fatalf("Run() error = %q, want PCK underflow", err)
	}
}

func TestRunWidePSH(t *testing.T) {
	spec := testSpec()
	const big = uint64(0xDEADBEEFCAFE)
	prog := []uint64{
		w(0x01, big), // PSH 0xDEADBEEFCAFE
		w(0x00, 0),   // HLT
	}

	m, err := NewMachine(spec, prog)
	if err != nil {
		t.Fatalf("NewMachine() error = %v", err)
	}
	trace, err := m.Run(8)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	last := trace[len(trace)-1]
	if last.Stack[0] != big {
		t.Fatalf("stack[0] = 0x%X, want 0x%X", last.Stack[0], big)
	}
}

func TestRunSnapshotBeyondWindow(t *testing.T) {
	// Regression: snapshot was hard-capped at 8 cells; values pushed to
	// stack positions >= 8 appeared as 0 in pure-trace output.
	spec := testSpec()
	prog := make([]uint64, 0, 11)
	for i := 1; i <= 9; i++ {
		prog = append(prog, w(0x01, uint64(i))) // PSH 1..9
	}
	prog = append(prog, w(0x00, 0)) // HLT

	m, err := NewMachine(spec, prog)
	if err != nil {
		t.Fatalf("NewMachine() error = %v", err)
	}
	trace, err := m.Run(16)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	last := trace[len(trace)-1]
	if last.SP != 9 {
		t.Fatalf("SP = %d, want 9", last.SP)
	}
	if len(last.Stack) != 9 {
		t.Fatalf("len(Stack) = %d, want 9", len(last.Stack))
	}
	for i := 0; i < 9; i++ {
		if got, want := last.Stack[i], uint64(i+1); got != want {
			t.Fatalf("Stack[%d] = %d, want %d", i, got, want)
		}
	}
}
