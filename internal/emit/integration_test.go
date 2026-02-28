// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit_test

import (
	"bytes"
	"strings"
	"testing"

	"lvdl-vm/internal/emit"
	"lvdl-vm/internal/lvdl"
	"lvdl-vm/internal/profile"
)

func runtime8Spec() *lvdl.Spec {
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

	return &lvdl.Spec{
		Machine: lvdl.Machine{
			WordBits: 16,
			Regs: []lvdl.Reg{
				{Name: "PC", Bits: 5},
				{Name: "SP", Bits: 4},
			},
			Stacks: []lvdl.Stack{{Name: "STK", Words: 8, Bits: 8}},
		},
		ISA: lvdl.ISA{
			Encoding: lvdl.Encoding{OpcodeBits: 8, OperandBits: 8},
			ByName:   byName,
			ByOpcode: byOpcode,
		},
	}
}

func runtime8TinySpec() *lvdl.Spec {
	spec := runtime8Spec()
	for i := range spec.Machine.Regs {
		if spec.Machine.Regs[i].Name == "PC" {
			spec.Machine.Regs[i].Bits = 4
		}
	}
	return spec
}

func w(op, imm uint8) uint16 {
	return (uint16(op) << 8) | uint16(imm)
}

func TestRuntime8EmitPrograms(t *testing.T) {
	spec := runtime8Spec()

	cases := []struct {
		name string
		prog []uint16
	}{
		{
			name: "add_2_3",
			prog: []uint16{w(0x01, 2), w(0x01, 3), w(0x04, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "add_overflow",
			prog: []uint16{w(0x01, 250), w(0x01, 10), w(0x04, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "sub_demo",
			prog: []uint16{w(0x01, 7), w(0x01, 2), w(0x03, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "and_demo",
			prog: []uint16{w(0x01, 0xAA), w(0x01, 0xCC), w(0x05, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "xor_demo",
			prog: []uint16{w(0x01, 0xAA), w(0x01, 0xCC), w(0x06, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "or_demo",
			prog: []uint16{w(0x01, 0xA0), w(0x01, 0x0F), w(0x07, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "dup_demo",
			prog: []uint16{w(0x01, 42), w(0x0E, 0), w(0x04, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "swp_demo",
			prog: []uint16{w(0x01, 7), w(0x01, 2), w(0x0F, 0), w(0x03, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "ovr_demo",
			prog: []uint16{w(0x01, 4), w(0x01, 3), w(0x10, 0), w(0x04, 0), w(0x04, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "inc_demo",
			prog: []uint16{w(0x01, 255), w(0x11, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "dec_demo",
			prog: []uint16{w(0x01, 0), w(0x12, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "cll_ret_demo",
			prog: []uint16{w(0x13, 4), w(0x01, 7), w(0x08, 0), w(0x00, 0), w(0x14, 0)},
		},
		{
			name: "pop_demo",
			prog: []uint16{w(0x01, 9), w(0x02, 0), w(0x01, 5), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "not_demo",
			prog: []uint16{w(0x01, 0x55), w(0x09, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "shl_demo",
			prog: []uint16{w(0x01, 0xAA), w(0x0A, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "shr_demo",
			prog: []uint16{w(0x01, 0xAA), w(0x0B, 0), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "jmp_demo",
			prog: []uint16{w(0x01, 1), w(0x0C, 3), w(0x01, 99), w(0x08, 0), w(0x00, 0)},
		},
		{
			name: "jnz_demo",
			prog: []uint16{
				w(0x01, 0), w(0x0D, 4), w(0x01, 1),
				w(0x0C, 5), w(0x01, 99),
				w(0x01, 1), w(0x0D, 9), w(0x01, 99),
				w(0x0C, 10), w(0x01, 2),
				w(0x04, 0), w(0x08, 0), w(0x00, 0),
			},
		},
		{
			name: "isa_smoke",
			prog: []uint16{
				w(0x13, 28),
				w(0x01, 2), w(0x01, 3), w(0x04, 0),
				w(0x01, 0xF0), w(0x05, 0),
				w(0x01, 0xAA), w(0x06, 0),
				w(0x01, 0x0F), w(0x07, 0),
				w(0x09, 0),
				w(0x0A, 0),
				w(0x0B, 0),
				w(0x11, 0),
				w(0x12, 0),
				w(0x0E, 0),
				w(0x0F, 0),
				w(0x10, 0),
				w(0x02, 0),
				w(0x05, 0),
				w(0x0C, 22),
				w(0x01, 99),
				w(0x0D, 24),
				w(0x01, 88),
				w(0x01, 0),
				w(0x03, 0),
				w(0x08, 0),
				w(0x00, 0),
				w(0x14, 0),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := emit.WriteLivePureHTML(&buf, "test", spec, tc.prog); err != nil {
				t.Fatalf("WriteLivePureHTML() error = %v", err)
			}

			got := buf.String()
			if strings.Contains(got, "<script") {
				t.Fatal("pure css output must not contain script")
			}
			if !strings.Contains(got, "wz-dispatch") {
				t.Fatal("output missing write-zone dispatch controls")
			}
			if strings.Contains(got, "\n") {
				t.Fatal("output is not fully minified (contains newlines)")
			}
			if !strings.Contains(got, "c10") || !strings.Contains(got, "u00") {
				t.Fatal("output missing carry/sum ALU staging state")
			}
		})
	}
}

func TestRuntime8EmitClockProgram(t *testing.T) {
	spec := runtime8Spec()
	prog := []uint16{w(0x01, 2), w(0x01, 3), w(0x04, 0), w(0x08, 0), w(0x00, 0)}

	var buf bytes.Buffer
	if err := emit.WriteLiveClockHTML(&buf, "test", spec, prog); err != nil {
		t.Fatalf("WriteLiveClockHTML() error = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "<script") {
		t.Fatal("clock output missing script")
	}
	if strings.Contains(got, ".checked=") || strings.Contains(got, ".checked =") {
		t.Fatal("clock script writes radio state directly")
	}
	if !strings.Contains(got, ".click();") {
		t.Fatal("clock script does not use click-only stepping")
	}
	for _, preset := range []string{"500", "250", "100", "10", "0"} {
		if !strings.Contains(got, `data-ms="`+preset+`"`) {
			t.Fatalf("missing speed preset %s", preset)
		}
	}
}

func TestRuntime8OptimizationToggle(t *testing.T) {
	spec := runtime8Spec()
	prog := []uint16{w(0x01, 2), w(0x01, 3), w(0x04, 0), w(0x08, 0), w(0x00, 0)}

	var pruned bytes.Buffer
	if err := emit.WriteLivePureHTMLWithOptions(
		&pruned,
		"test",
		spec,
		prog,
		emit.Options{Optimization: emit.OptimizationControlState},
	); err != nil {
		t.Fatalf("WriteLivePureHTMLWithOptions(control-state) error = %v", err)
	}

	var full bytes.Buffer
	if err := emit.WriteLivePureHTMLWithOptions(
		&full,
		"test",
		spec,
		prog,
		emit.Options{Optimization: emit.OptimizationNone},
	); err != nil {
		t.Fatalf("WriteLivePureHTMLWithOptions(none) error = %v", err)
	}

	if len(pruned.String()) >= len(full.String()) {
		t.Fatalf("expected pruned output to be smaller: pruned=%d full=%d", len(pruned.String()), len(full.String()))
	}
	if strings.Count(pruned.String(), `class="stk-row"`) >= strings.Count(full.String(), `class="stk-row"`) {
		t.Fatal("expected pruned output to render fewer stack rows")
	}
	if strings.Count(pruned.String(), `class="pl" data-p=`) >= strings.Count(full.String(), `class="pl" data-p=`) {
		t.Fatal("expected pruned output to render fewer program rows")
	}
}

func TestRuntime8TinyProfileProgramBounds(t *testing.T) {
	spec := runtime8TinySpec()
	prog := []uint16{w(0x01, 2), w(0x01, 3), w(0x04, 0), w(0x08, 0), w(0x00, 0)}

	var buf bytes.Buffer
	err := emit.WriteLivePureHTMLWithOptions(
		&buf,
		"tiny",
		spec,
		prog,
		emit.Options{
			Optimization: emit.OptimizationNone,
			Profile:      profile.Runtime8Tiny,
		},
	)
	if err != nil {
		t.Fatalf("WriteLivePureHTMLWithOptions(runtime8-tiny) error = %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `class="pl" data-p="15"`) {
		t.Fatal("expected tiny profile to render rows up to pc=15")
	}
	if strings.Contains(got, `class="pl" data-p="16"`) {
		t.Fatal("tiny profile should not render rows beyond pc=15")
	}
}

func TestRuntime8TinyProfileRejectsProgramOverflow(t *testing.T) {
	spec := runtime8TinySpec()
	prog := make([]uint16, 17)
	prog[16] = w(0x00, 0)

	var buf bytes.Buffer
	err := emit.WriteLivePureHTMLWithOptions(
		&buf,
		"tiny",
		spec,
		prog,
		emit.Options{
			Optimization: emit.OptimizationNone,
			Profile:      profile.Runtime8Tiny,
		},
	)
	if err == nil {
		t.Fatal("expected profile overflow error for 17-word program")
	}
	if !strings.Contains(err.Error(), "supports up to 16 words") {
		t.Fatalf("error = %q, want supports up to 16 words", err)
	}
}

func TestRuntime8PhaseRadiosCompactedToProgramOps(t *testing.T) {
	spec := runtime8Spec()
	prog := []uint16{w(0x01, 2), w(0x01, 3), w(0x04, 0), w(0x08, 0), w(0x00, 0)}

	var buf bytes.Buffer
	if err := emit.WriteLivePureHTML(&buf, "test", spec, prog); err != nil {
		t.Fatalf("WriteLivePureHTML() error = %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `id="h0"`) ||
		!strings.Contains(got, `id="h1"`) ||
		!strings.Contains(got, `id="h20"`) ||
		!strings.Contains(got, `id="h60"`) {
		t.Fatal("missing expected active phase radios")
	}
	if strings.Contains(got, `id="h250"`) || strings.Contains(got, `id="h300"`) {
		t.Fatal("found phase radios for instructions not present in program")
	}
}
