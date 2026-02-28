// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"lvdl-vm/internal/lvdl"
)

func testRuntime8Spec() *lvdl.Spec {
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

func tw(op, imm uint8) uint16 { return (uint16(op) << 8) | uint16(imm) }

func TestWriteLivePureHTMLNoScriptAndStateShape(t *testing.T) {
	spec := testRuntime8Spec()
	prog := []uint16{tw(0x01, 2), tw(0x01, 3), tw(0x04, 0), tw(0x08, 0), tw(0x00, 0)}

	var buf bytes.Buffer
	if err := WriteLivePureHTML(&buf, "test", spec, prog); err != nil {
		t.Fatalf("WriteLivePureHTML() error = %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "<script") {
		t.Fatal("found <script> tag in pure css output")
	}
	checks := []string{
		"id=\"k000\"",
		"id=\"k771\"",
		"id=\"u00\"",
		"id=\"c81\"",
		"id=\"os0\"",
		"id=\"op0\"",
		"wz-dispatch",
		"class=\"wz ",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Fatalf("missing expected runtime8 marker %q", c)
		}
	}
}

func TestWriteLiveClockHTMLClockScriptInvariant(t *testing.T) {
	spec := testRuntime8Spec()
	prog := []uint16{tw(0x01, 2), tw(0x01, 3), tw(0x04, 0), tw(0x08, 0), tw(0x00, 0)}

	var buf bytes.Buffer
	if err := WriteLiveClockHTML(&buf, "test", spec, prog); err != nil {
		t.Fatalf("WriteLiveClockHTML() error = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "<script") {
		t.Fatal("missing js clock script in clock mode output")
	}
	if strings.Contains(got, ".checked=") || strings.Contains(got, ".checked =") {
		t.Fatal("clock script must not write radio state directly")
	}
	if !strings.Contains(got, ".click();") {
		t.Fatal("clock script must use click-only stepping")
	}
	if !strings.Contains(got, "querySelector('.program')") || !strings.Contains(got, "scrollTop") {
		t.Fatal("clock script missing program auto-scroll behavior")
	}
	if !strings.Contains(got, "data-ms=\"500\"") ||
		!strings.Contains(got, "data-ms=\"250\"") ||
		!strings.Contains(got, "data-ms=\"100\"") ||
		!strings.Contains(got, "data-ms=\"10\"") ||
		!strings.Contains(got, "data-ms=\"0\"") {
		t.Fatal("clock controls missing required speed presets")
	}
}

func TestWriteLivePureHTMLRejectsUnsupportedOpcode(t *testing.T) {
	spec := testRuntime8Spec()
	prog := []uint16{tw(0xFE, 0), tw(0x00, 0)}

	var buf bytes.Buffer
	err := WriteLivePureHTML(&buf, "test", spec, prog)
	if err == nil {
		t.Fatal("WriteLivePureHTML() error = nil, want unknown opcode error")
	}
	if !strings.Contains(err.Error(), "unknown opcode") {
		t.Fatalf("error = %q, want unknown opcode", err)
	}
}

type wzRule struct {
	class   string
	must    []string
	mustNot []string
}

type htmlRuntime struct {
	checked       map[string]bool
	inputNameByID map[string]string
	idsByName     map[string][]string
	targetByClass map[string]string
	rules         []wzRule
}

func parseRuntimeHTML(t *testing.T, html string) *htmlRuntime {
	t.Helper()

	rt := &htmlRuntime{
		checked:       make(map[string]bool),
		inputNameByID: make(map[string]string),
		idsByName:     make(map[string][]string),
		targetByClass: make(map[string]string),
	}

	inputRe := regexp.MustCompile(`<input type="radio" name="([^"]+)" id="([^"]+)"([^>]*)>`)
	for _, m := range inputRe.FindAllStringSubmatch(html, -1) {
		name := m[1]
		id := m[2]
		attr := m[3]
		rt.inputNameByID[id] = name
		rt.idsByName[name] = append(rt.idsByName[name], id)
		if strings.Contains(attr, " checked") {
			rt.checked[id] = true
		}
	}

	labelRe := regexp.MustCompile(`<div class="([^"]*)"><label for="([^"]+)">`)
	for _, m := range labelRe.FindAllStringSubmatch(html, -1) {
		classes := strings.Fields(m[1])
		target := m[2]
		for _, class := range classes {
			if !strings.HasPrefix(class, "wz") {
				continue
			}
			switch class {
			case "wz", "wz-dispatch", "wz-set", "wz-next", "wz-reset":
				continue
			default:
				rt.targetByClass[class] = target
			}
		}
	}

	styleRe := regexp.MustCompile(`(?s)<style>(.*)</style>`)
	styleM := styleRe.FindStringSubmatch(html)
	if len(styleM) != 2 {
		t.Fatal("missing <style> block in generated HTML")
	}
	css := styleM[1]

	ruleRe := regexp.MustCompile(`(?s)([^{}]+)\{([^{}]+)\}`)
	classRe := regexp.MustCompile(`\.(wz[A-Za-z0-9_-]*)\s*$`)
	notHasRe := regexp.MustCompile(`:not\(:has\(#([A-Za-z0-9_]+):checked\)\)`)
	hasRe := regexp.MustCompile(`:has\(#([A-Za-z0-9_]+):checked\)`)

	for _, m := range ruleRe.FindAllStringSubmatch(css, -1) {
		selector := strings.TrimSpace(m[1])
		decl := m[2]
		if !strings.Contains(selector, ".wz") || !strings.Contains(decl, "visibility:visible") {
			continue
		}

		classM := classRe.FindStringSubmatch(selector)
		if len(classM) != 2 {
			continue
		}
		class := classM[1]
		switch class {
		case "wz", "wz-dispatch", "wz-set", "wz-next", "wz-reset":
			continue
		}

		negMatches := notHasRe.FindAllStringSubmatch(selector, -1)
		mustNot := make([]string, 0, len(negMatches))
		for _, nm := range negMatches {
			mustNot = append(mustNot, nm[1])
		}

		noNot := notHasRe.ReplaceAllString(selector, "")
		posMatches := hasRe.FindAllStringSubmatch(noNot, -1)
		must := make([]string, 0, len(posMatches))
		for _, pm := range posMatches {
			must = append(must, pm[1])
		}

		rt.rules = append(rt.rules, wzRule{
			class:   class,
			must:    must,
			mustNot: mustNot,
		})
	}

	return rt
}

func (rt *htmlRuntime) click(id string) error {
	name, ok := rt.inputNameByID[id]
	if !ok {
		return fmt.Errorf("unknown radio id %q", id)
	}
	for _, other := range rt.idsByName[name] {
		delete(rt.checked, other)
	}
	rt.checked[id] = true
	return nil
}

func (rt *htmlRuntime) visibleActions() []string {
	out := make([]string, 0, 2)
	seen := map[string]bool{}

	for _, rule := range rt.rules {
		if _, ok := rt.targetByClass[rule.class]; !ok {
			continue
		}
		match := true
		for _, id := range rule.must {
			if !rt.checked[id] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		for _, id := range rule.mustNot {
			if rt.checked[id] {
				match = false
				break
			}
		}
		if !match || seen[rule.class] {
			continue
		}
		seen[rule.class] = true
		out = append(out, rule.class)
	}
	return out
}

func (rt *htmlRuntime) outputValue() (int, bool) {
	if !rt.checked[outValidID(1)] {
		return 0, false
	}
	val := 0
	for bit := 0; bit < 8; bit++ {
		if rt.checked[outBitID(bit, 1)] {
			val |= 1 << bit
		}
	}
	return val, true
}

func (rt *htmlRuntime) pcValue() int {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if rt.checked[pcID(pc)] {
			return pc
		}
	}
	return -1
}

func (rt *htmlRuntime) spValue() int {
	for sp := 0; sp <= runtimeStackSize; sp++ {
		if rt.checked[spID(sp)] {
			return sp
		}
	}
	return -1
}

func (rt *htmlRuntime) stackCellValue(cell int) int {
	val := 0
	for bit := 0; bit < runtimeWordBits; bit++ {
		if rt.checked[stkBitID(cell, bit, 1)] {
			val |= 1 << bit
		}
	}
	return val
}

type runtimeExecResult struct {
	halted          bool
	deterministic   bool
	steps           int
	conflictStep    int
	conflictActions []string
	output          int
	outputValid     bool
	pc              int
	sp              int
	top             int
}

func runToHalt(t *testing.T, html string) runtimeExecResult {
	t.Helper()
	rt := parseRuntimeHTML(t, html)
	res := runtimeExecResult{
		deterministic: true,
		conflictStep:  -1,
	}

	const maxSteps = 2000
	for step := 0; step < maxSteps; step++ {
		actions := rt.visibleActions()
		if len(actions) == 0 {
			res.halted = true
			res.steps = step
			break
		}
		if len(actions) != 1 {
			res.deterministic = false
			res.conflictStep = step
			res.conflictActions = actions
			res.steps = step
			break
		}

		target := rt.targetByClass[actions[0]]
		if err := rt.click(target); err != nil {
			t.Fatalf("click(%q) failed at step %d: %v", target, step, err)
		}
	}
	if !res.halted && res.deterministic {
		t.Fatalf("runtime did not halt within %d steps", maxSteps)
	}
	res.output, res.outputValid = rt.outputValue()
	res.pc = rt.pcValue()
	res.sp = rt.spValue()
	if res.sp > 0 {
		res.top = rt.stackCellValue(res.sp - 1)
	}
	return res
}

func runtimeCases() []struct {
	name    string
	prog    []uint16
	want    int
	wantOut bool
	wantPC  int
	wantSP  int
	wantTop int
} {
	return []struct {
		name    string
		prog    []uint16
		want    int
		wantOut bool
		wantPC  int
		wantSP  int
		wantTop int
	}{
		{
			name:    "add_2_3",
			prog:    []uint16{tw(0x01, 2), tw(0x01, 3), tw(0x04, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    5,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 5,
		},
		{
			name:    "add_wrap_250_10",
			prog:    []uint16{tw(0x01, 250), tw(0x01, 10), tw(0x04, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    4,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 4,
		},
		{
			name:    "sub_7_2",
			prog:    []uint16{tw(0x01, 7), tw(0x01, 2), tw(0x03, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    5,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 5,
		},
		{
			name:    "and_aa_cc",
			prog:    []uint16{tw(0x01, 0xAA), tw(0x01, 0xCC), tw(0x05, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    0x88,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 0x88,
		},
		{
			name:    "xor_aa_cc",
			prog:    []uint16{tw(0x01, 0xAA), tw(0x01, 0xCC), tw(0x06, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    0x66,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 0x66,
		},
		{
			name:    "or_a0_0f",
			prog:    []uint16{tw(0x01, 0xA0), tw(0x01, 0x0F), tw(0x07, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    0xAF,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 0xAF,
		},
		{
			name:    "dup_42_add",
			prog:    []uint16{tw(0x01, 42), tw(0x0E, 0), tw(0x04, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    84,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 84,
		},
		{
			name:    "swp_7_2_sub",
			prog:    []uint16{tw(0x01, 7), tw(0x01, 2), tw(0x0F, 0), tw(0x03, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    251,
			wantOut: true,
			wantPC:  5,
			wantSP:  1,
			wantTop: 251,
		},
		{
			name:    "ovr_4_3_add_add",
			prog:    []uint16{tw(0x01, 4), tw(0x01, 3), tw(0x10, 0), tw(0x04, 0), tw(0x04, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    11,
			wantOut: true,
			wantPC:  6,
			wantSP:  1,
			wantTop: 11,
		},
		{
			name:    "inc_wrap_255",
			prog:    []uint16{tw(0x01, 255), tw(0x11, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    0,
			wantOut: true,
			wantPC:  3,
			wantSP:  1,
			wantTop: 0,
		},
		{
			name:    "dec_wrap_0",
			prog:    []uint16{tw(0x01, 0), tw(0x12, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    255,
			wantOut: true,
			wantPC:  3,
			wantSP:  1,
			wantTop: 255,
		},
		{
			name:    "cll_ret_basic",
			prog:    []uint16{tw(0x13, 4), tw(0x01, 7), tw(0x08, 0), tw(0x00, 0), tw(0x14, 0)},
			want:    7,
			wantOut: true,
			wantPC:  3,
			wantSP:  1,
			wantTop: 7,
		},
		{
			name: "cll_jump_visible",
			prog: []uint16{
				tw(0x13, 3), // CLL fn
				tw(0x08, 0), // SAY (would print return address if CLL does not jump)
				tw(0x00, 0), // HLT
				tw(0x01, 7), // fn: PSH 7
				tw(0x0F, 0), // SWP (move return address back to TOS for RET)
				tw(0x14, 0), // RET
			},
			want:    7,
			wantOut: true,
			wantPC:  2,
			wantSP:  1,
			wantTop: 7,
		},
		{
			name:    "pop_discard",
			prog:    []uint16{tw(0x01, 9), tw(0x02, 0), tw(0x01, 5), tw(0x08, 0), tw(0x00, 0)},
			want:    5,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 5,
		},
		{
			name:    "not_55",
			prog:    []uint16{tw(0x01, 0x55), tw(0x09, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    0xAA,
			wantOut: true,
			wantPC:  3,
			wantSP:  1,
			wantTop: 0xAA,
		},
		{
			name:    "shl_aa",
			prog:    []uint16{tw(0x01, 0xAA), tw(0x0A, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    0x54,
			wantOut: true,
			wantPC:  3,
			wantSP:  1,
			wantTop: 0x54,
		},
		{
			name:    "shr_aa",
			prog:    []uint16{tw(0x01, 0xAA), tw(0x0B, 0), tw(0x08, 0), tw(0x00, 0)},
			want:    0x55,
			wantOut: true,
			wantPC:  3,
			wantSP:  1,
			wantTop: 0x55,
		},
		{
			name:    "jmp_skip",
			prog:    []uint16{tw(0x01, 1), tw(0x0C, 3), tw(0x01, 99), tw(0x08, 0), tw(0x00, 0)},
			want:    1,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 1,
		},
		{
			name:    "jnz_taken",
			prog:    []uint16{tw(0x01, 1), tw(0x0D, 3), tw(0x01, 99), tw(0x08, 0), tw(0x00, 0)},
			want:    1,
			wantOut: true,
			wantPC:  4,
			wantSP:  1,
			wantTop: 1,
		},
		{
			name:    "jnz_not_taken",
			prog:    []uint16{tw(0x01, 0), tw(0x0D, 3), tw(0x01, 7), tw(0x08, 0), tw(0x00, 0)},
			want:    7,
			wantOut: true,
			wantPC:  4,
			wantSP:  2,
			wantTop: 7,
		},
		{
			name: "jnz_demo",
			prog: []uint16{
				tw(0x01, 0), tw(0x0D, 4), tw(0x01, 1),
				tw(0x0C, 5), tw(0x01, 99),
				tw(0x01, 1), tw(0x0D, 9), tw(0x01, 99),
				tw(0x0C, 10), tw(0x01, 2),
				tw(0x04, 0), tw(0x08, 0), tw(0x00, 0),
			},
			want:    3,
			wantOut: true,
			wantPC:  12,
			wantSP:  3,
			wantTop: 3,
		},
		{
			name: "isa_smoke",
			prog: []uint16{
				tw(0x13, 28),
				tw(0x01, 2), tw(0x01, 3), tw(0x04, 0),
				tw(0x01, 0xF0), tw(0x05, 0),
				tw(0x01, 0xAA), tw(0x06, 0),
				tw(0x01, 0x0F), tw(0x07, 0),
				tw(0x09, 0),
				tw(0x0A, 0),
				tw(0x0B, 0),
				tw(0x11, 0),
				tw(0x12, 0),
				tw(0x0E, 0),
				tw(0x0F, 0),
				tw(0x10, 0),
				tw(0x02, 0),
				tw(0x05, 0),
				tw(0x0C, 22),
				tw(0x01, 99),
				tw(0x0D, 24),
				tw(0x01, 88),
				tw(0x01, 0),
				tw(0x03, 0),
				tw(0x08, 0),
				tw(0x00, 0),
				tw(0x14, 0),
			},
			want:    0x50,
			wantOut: true,
			wantPC:  27,
			wantSP:  1,
			wantTop: 0x50,
		},
	}
}

func TestWriteLivePureHTMLRuntimeProgramsOutputAndState(t *testing.T) {
	spec := testRuntime8Spec()
	cases := runtimeCases()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteLivePureHTML(&buf, "test", spec, tc.prog); err != nil {
				t.Fatalf("WriteLivePureHTML() error = %v", err)
			}

			got := runToHalt(t, buf.String())
			if !got.deterministic {
				t.Fatalf(
					"non-deterministic runtime at step %d: %v",
					got.conflictStep, got.conflictActions,
				)
			}
			if !got.halted {
				t.Fatal("runtime did not halt")
			}
			if got.outputValid != tc.wantOut || got.output != tc.want {
				t.Fatalf("output = (%d,%v), want (%d,%v)", got.output, got.outputValid, tc.want, tc.wantOut)
			}
			if got.sp != tc.wantSP || got.pc != tc.wantPC || got.top != tc.wantTop {
				t.Fatalf(
					"state = (pc=%d sp=%d top=%d), want (pc=%d sp=%d top=%d)",
					got.pc, got.sp, got.top, tc.wantPC, tc.wantSP, tc.wantTop,
				)
			}
		})
	}
}

func TestWriteLivePureHTMLRuntimeDeterministicSingleAction(t *testing.T) {
	spec := testRuntime8Spec()
	for _, tc := range runtimeCases() {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteLivePureHTML(&buf, "test", spec, tc.prog); err != nil {
				t.Fatalf("WriteLivePureHTML() error = %v", err)
			}
			got := runToHalt(t, buf.String())
			if !got.deterministic {
				t.Fatalf(
					"expected single visible write-zone each step; conflict at step %d: %v",
					got.conflictStep, got.conflictActions,
				)
			}
		})
	}
}

func TestWriteLivePureHTMLOptModesEquivalent(t *testing.T) {
	spec := testRuntime8Spec()
	for _, tc := range runtimeCases() {
		t.Run(tc.name, func(t *testing.T) {
			var pruned bytes.Buffer
			if err := WriteLivePureHTMLWithOptions(
				&pruned,
				"test",
				spec,
				tc.prog,
				Options{Optimization: OptimizationControlState},
			); err != nil {
				t.Fatalf("WriteLivePureHTMLWithOptions(control-state) error = %v", err)
			}

			var full bytes.Buffer
			if err := WriteLivePureHTMLWithOptions(
				&full,
				"test",
				spec,
				tc.prog,
				Options{Optimization: OptimizationNone},
			); err != nil {
				t.Fatalf("WriteLivePureHTMLWithOptions(none) error = %v", err)
			}

			gotPruned := runToHalt(t, pruned.String())
			gotFull := runToHalt(t, full.String())

			if gotPruned.deterministic != gotFull.deterministic ||
				gotPruned.halted != gotFull.halted ||
				gotPruned.outputValid != gotFull.outputValid ||
				gotPruned.output != gotFull.output ||
				gotPruned.pc != gotFull.pc ||
				gotPruned.sp != gotFull.sp ||
				gotPruned.top != gotFull.top {
				t.Fatalf(
					"opt mismatch:\npruned=%+v\nfull=%+v",
					gotPruned, gotFull,
				)
			}
		})
	}
}

func makeProgram(words ...uint16) []uint16 {
	prog := make([]uint16, runtimeProgWords)
	for i := range prog {
		prog[i] = tw(0x00, 0)
	}
	copy(prog, words)
	return prog
}

func TestWriteLivePureHTMLStackUnderflowProgramsHalt(t *testing.T) {
	spec := testRuntime8Spec()
	cases := []struct {
		name string
		prog []uint16
	}{
		{name: "add_underflow", prog: makeProgram(tw(0x04, 0), tw(0x00, 0))},
		{name: "pop_underflow", prog: makeProgram(tw(0x02, 0), tw(0x00, 0))},
		{name: "jnz_underflow", prog: makeProgram(tw(0x0D, 0), tw(0x00, 0))},
		{name: "ret_underflow", prog: makeProgram(tw(0x14, 0), tw(0x00, 0))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteLivePureHTML(&buf, "test", spec, tc.prog); err != nil {
				t.Fatalf("WriteLivePureHTML() error = %v", err)
			}
			got := runToHalt(t, buf.String())
			if !got.deterministic {
				t.Fatalf("expected deterministic halt, conflict at step %d", got.conflictStep)
			}
			if !got.halted || got.steps != 0 {
				t.Fatalf("expected halt at step 0, got halted=%v steps=%d", got.halted, got.steps)
			}
			if got.pc != 0 || got.sp != 0 {
				t.Fatalf("expected control state pc=0 sp=0, got pc=%d sp=%d", got.pc, got.sp)
			}
			if got.outputValid {
				t.Fatalf("unexpected output latch set: %d", got.output)
			}
		})
	}
}

func TestWriteLivePureHTMLControlFlowUpperBoundPrograms(t *testing.T) {
	spec := testRuntime8Spec()
	cases := []struct {
		name    string
		prog    []uint16
		wantPC  int
		wantSP  int
		wantOut bool
		wantVal int
	}{
		{
			name:    "jmp_to_pc31",
			prog:    makeProgram(tw(0x0C, 31)),
			wantPC:  31,
			wantSP:  0,
			wantOut: false,
		},
		{
			name:    "jnz_taken_to_pc31",
			prog:    makeProgram(tw(0x01, 1), tw(0x0D, 31)),
			wantPC:  31,
			wantSP:  1,
			wantOut: false,
		},
		{
			name:    "jnz_not_taken_fallthrough",
			prog:    makeProgram(tw(0x01, 0), tw(0x0D, 31), tw(0x01, 7), tw(0x08, 0), tw(0x00, 0)),
			wantPC:  4,
			wantSP:  2,
			wantOut: true,
			wantVal: 7,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteLivePureHTML(&buf, "test", spec, tc.prog); err != nil {
				t.Fatalf("WriteLivePureHTML() error = %v", err)
			}
			got := runToHalt(t, buf.String())
			if !got.halted || !got.deterministic {
				t.Fatalf("expected deterministic halt, got halted=%v deterministic=%v", got.halted, got.deterministic)
			}
			if got.pc != tc.wantPC || got.sp != tc.wantSP {
				t.Fatalf("state = (pc=%d sp=%d), want (pc=%d sp=%d)", got.pc, got.sp, tc.wantPC, tc.wantSP)
			}
			if got.outputValid != tc.wantOut || (tc.wantOut && got.output != tc.wantVal) {
				t.Fatalf("output = (%d,%v), want (%d,%v)", got.output, got.outputValid, tc.wantVal, tc.wantOut)
			}
		})
	}
}
