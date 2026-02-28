// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package asm

import (
	"strings"
	"testing"

	"lvdl-vm/internal/lvdl"
)

func testISA() lvdl.ISA {
	instrs := []struct {
		name string
		op   uint8
	}{
		{"HLT", 0x00},
		{"PSH", 0x01},
		{"POP", 0x02},
		{"ADD", 0x04},
		{"SUB", 0x03},
		{"AND", 0x05},
		{"XOR", 0x06},
		{"OR", 0x07},
		{"DUP", 0x0E},
		{"SWP", 0x0F},
		{"OVR", 0x10},
		{"INC", 0x11},
		{"DEC", 0x12},
		{"CLL", 0x13},
		{"RET", 0x14},
		{"NOT", 0x09},
		{"SHL", 0x0A},
		{"SHR", 0x0B},
		{"JMP", 0x0C},
		{"JNZ", 0x0D},
		{"SAY", 0x08},
	}

	byName := make(map[string]lvdl.Instr, len(instrs))
	byOpcode := make(map[uint8]lvdl.Instr, len(instrs))
	for _, in := range instrs {
		ins := lvdl.Instr{Name: in.name, Opcode: in.op}
		byName[in.name] = ins
		byOpcode[in.op] = ins
	}
	return lvdl.ISA{ByName: byName, ByOpcode: byOpcode}
}

func w(op, imm uint8) uint16 {
	return (uint16(op) << 8) | uint16(imm)
}

func TestAssembleBasicProgram(t *testing.T) {
	src := strings.NewReader("psh 2\nPSH 3\nadd\nsay\nhlt\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	want := []uint16{w(0x01, 2), w(0x01, 3), w(0x04, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssemblePSHHex(t *testing.T) {
	src := strings.NewReader("PSH 0xFA\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if got.Words[0] != w(0x01, 0xFA) {
		t.Fatalf("words[0] = 0x%04X, want 0x%04X", got.Words[0], w(0x01, 0xFA))
	}
}

func TestAssemblePSHBinary(t *testing.T) {
	src := strings.NewReader("PSH 0b10101010\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if got.Words[0] != w(0x01, 0xAA) {
		t.Fatalf("words[0] = 0x%04X, want 0x%04X", got.Words[0], w(0x01, 0xAA))
	}
}

func TestAssemblePOP(t *testing.T) {
	src := strings.NewReader("PSH 9\nPOP\nPSH 5\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 9), w(0x02, 0), w(0x01, 5), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleAND(t *testing.T) {
	src := strings.NewReader("PSH 0b10101010\nPSH 0b11001100\nAND\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 0xAA), w(0x01, 0xCC), w(0x05, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleXOR(t *testing.T) {
	src := strings.NewReader("PSH 0b10101010\nPSH 0b11001100\nXOR\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 0xAA), w(0x01, 0xCC), w(0x06, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleOR(t *testing.T) {
	src := strings.NewReader("PSH 0b10100000\nPSH 0b00001111\nOR\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 0xA0), w(0x01, 0x0F), w(0x07, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleDUP(t *testing.T) {
	src := strings.NewReader("PSH 42\nDUP\nADD\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 42), w(0x0E, 0), w(0x04, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleSWP(t *testing.T) {
	src := strings.NewReader("PSH 7\nPSH 2\nSWP\nSUB\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 7), w(0x01, 2), w(0x0F, 0), w(0x03, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleOVR(t *testing.T) {
	src := strings.NewReader("PSH 4\nPSH 3\nOVR\nADD\nADD\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 4), w(0x01, 3), w(0x10, 0), w(0x04, 0), w(0x04, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleINC(t *testing.T) {
	src := strings.NewReader("PSH 255\nINC\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 255), w(0x11, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleDEC(t *testing.T) {
	src := strings.NewReader("PSH 0\nDEC\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 0), w(0x12, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleNOT(t *testing.T) {
	src := strings.NewReader("PSH 0b01010101\nNOT\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 0x55), w(0x09, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleSHL(t *testing.T) {
	src := strings.NewReader("PSH 0b10101010\nSHL\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 0xAA), w(0x0A, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleSHR(t *testing.T) {
	src := strings.NewReader("PSH 0b10101010\nSHR\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 0xAA), w(0x0B, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleSUB(t *testing.T) {
	src := strings.NewReader("PSH 7\nPSH 2\nSUB\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 7), w(0x01, 2), w(0x03, 0), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleJMP(t *testing.T) {
	src := strings.NewReader("JMP 12\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x0C, 12), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleJMPMaxTarget(t *testing.T) {
	src := strings.NewReader("JMP 31\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x0C, 31), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleJMPLabel(t *testing.T) {
	src := strings.NewReader("PSH 1\nJMP done\nPSH 99\ndone:\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 1), w(0x0C, 3), w(0x01, 99), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleJMPRangeError(t *testing.T) {
	src := strings.NewReader("JMP 32\nHLT\n")
	_, err := Assemble(src, testISA())
	if err == nil {
		t.Fatal("Assemble() error = nil, want JMP range error")
	}
	if !strings.Contains(err.Error(), "JMP target out of range") {
		t.Fatalf("Assemble() error = %q, want JMP range error", err)
	}
}

func TestAssembleJNZ(t *testing.T) {
	src := strings.NewReader("PSH 1\nJNZ 3\nPSH 99\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 1), w(0x0D, 3), w(0x01, 99), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleJNZMaxTarget(t *testing.T) {
	src := strings.NewReader("PSH 1\nJNZ 31\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 1), w(0x0D, 31), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleCLLLabelAndRET(t *testing.T) {
	src := strings.NewReader("CLL fn\nPSH 7\nSAY\nHLT\nfn:\nRET\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x13, 4), w(0x01, 7), w(0x08, 0), w(0x00, 0), w(0x14, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleCLLRangeError(t *testing.T) {
	src := strings.NewReader("CLL 32\nHLT\n")
	_, err := Assemble(src, testISA())
	if err == nil {
		t.Fatal("Assemble() error = nil, want CLL range error")
	}
	if !strings.Contains(err.Error(), "CLL target out of range") {
		t.Fatalf("Assemble() error = %q, want CLL range error", err)
	}
}

func TestAssembleJNZLabel(t *testing.T) {
	src := strings.NewReader("PSH 1\nJNZ out\nPSH 99\nout:\nSAY\nHLT\n")
	got, err := Assemble(src, testISA())
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	want := []uint16{w(0x01, 1), w(0x0D, 3), w(0x01, 99), w(0x08, 0), w(0x00, 0)}
	if len(got.Words) != len(want) {
		t.Fatalf("len(words) = %d, want %d", len(got.Words), len(want))
	}
	for i := range want {
		if got.Words[i] != want[i] {
			t.Fatalf("words[%d] = 0x%04X, want 0x%04X", i, got.Words[i], want[i])
		}
	}
}

func TestAssembleUnknownJumpLabel(t *testing.T) {
	src := strings.NewReader("JMP nowhere\nHLT\n")
	_, err := Assemble(src, testISA())
	if err == nil {
		t.Fatal("Assemble() error = nil, want bad jump target")
	}
	if !strings.Contains(err.Error(), "bad jump target") {
		t.Fatalf("Assemble() error = %q, want bad jump target", err)
	}
}

func TestAssembleJNZRangeError(t *testing.T) {
	src := strings.NewReader("JNZ 32\nHLT\n")
	_, err := Assemble(src, testISA())
	if err == nil {
		t.Fatal("Assemble() error = nil, want JNZ range error")
	}
	if !strings.Contains(err.Error(), "JNZ target out of range") {
		t.Fatalf("Assemble() error = %q, want JNZ range error", err)
	}
}

func TestAssemblePSHRangeError(t *testing.T) {
	src := strings.NewReader("PSH 256\nHLT\n")
	_, err := Assemble(src, testISA())
	if err == nil {
		t.Fatal("Assemble() error = nil, want imm8 range error")
	}
	if !strings.Contains(err.Error(), "bad imm8") {
		t.Fatalf("Assemble() error = %q, want bad imm8", err)
	}
}

func TestAssembleUnsupportedInstruction(t *testing.T) {
	src := strings.NewReader("FOO 5\nHLT\n")
	_, err := Assemble(src, testISA())
	if err == nil {
		t.Fatal("Assemble() error = nil, want unsupported instruction error")
	}
	if !strings.Contains(err.Error(), "supports only HLT/PSH/POP/ADD/SUB/AND/XOR/OR/DUP/SWP/OVR/INC/DEC/CLL/RET/NOT/SHL/SHR/JMP/JNZ/SAY") {
		t.Fatalf("Assemble() error = %q, want supported-set message", err)
	}
}

func TestAssembleNoOperandInstructionRejectsArg(t *testing.T) {
	src := strings.NewReader("ADD 1\nHLT\n")
	_, err := Assemble(src, testISA())
	if err == nil {
		t.Fatal("Assemble() error = nil, want operand rejection")
	}
	if !strings.Contains(err.Error(), "takes no operand") {
		t.Fatalf("Assemble() error = %q, want takes no operand", err)
	}
}

func TestAssembleWithOptionsProgramBound(t *testing.T) {
	src := strings.NewReader("JMP 16\nHLT\n")
	_, err := AssembleWithOptions(src, testISA(), Options{MaxProgramPC: 15})
	if err == nil {
		t.Fatal("AssembleWithOptions() error = nil, want range error")
	}
	if !strings.Contains(err.Error(), "JMP target out of range (0..15)") {
		t.Fatalf("AssembleWithOptions() error = %q, want tiny-profile range message", err)
	}
}
