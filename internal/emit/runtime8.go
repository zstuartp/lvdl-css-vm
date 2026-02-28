// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"lvdl-vm/internal/lvdl"
	"lvdl-vm/internal/profile"
)

var (
	cssSpaceAroundTokenRe = regexp.MustCompile(`\s*([{}:;,>])\s*`)
	cssMultiSpaceRe       = regexp.MustCompile(`\s+`)
)

const (
	runtimeProgWords = 32
	runtimeStackSize = 8
	runtimeWordBits  = 8

	phaseDispatch = 0

	phasePSHStart = 1
	phasePSHEnd   = 11

	phaseADDStart = 20
	phaseADDEnd   = 47

	phaseDUPStart = 48
	phaseDUPEnd   = 59

	phaseSAYStart = 60
	phaseSAYEnd   = 70

	phasePOPStart = 71
	phasePOPEnd   = 73

	phaseANDStart = 80
	phaseANDEnd   = 98

	phaseXORStart = 100
	phaseXOREnd   = 118

	phaseORStart = 120
	phaseOREnd   = 138

	phaseNOTStart = 140
	phaseNOTEnd   = 157

	phaseSHLStart = 160
	phaseSHLEnd   = 177

	phaseSHRStart = 180
	phaseSHREnd   = 197

	phaseJMPStart = 200
	phaseJMPEnd   = 201

	phaseJNZStart = 210
	phaseJNZEnd   = 211

	phaseSUBStart = 220
	phaseSUBEnd   = 247

	phaseSWPStart = 250
	phaseSWPEnd   = 275

	phaseOVRStart = 280
	phaseOVREnd   = 290

	phaseINCStart = 300
	phaseINCEnd   = 326

	phaseDECStart = 330
	phaseDECEnd   = 356

	phaseCLLStart = 360
	phaseCLLEnd   = 370

	phaseRETStart = 380
	phaseRETEnd   = 382

	runtimeMaxPhase = phaseRETEnd
)

type decodedWord struct {
	Opcode  uint8
	Operand uint8
	Name    string
	Text    string
}

type optimizationPass string

const (
	passControlReachability optimizationPass = "control-reachability"
	passTrimProgramUI       optimizationPass = "trim-program-ui"
	passTrimStackUI         optimizationPass = "trim-stack-ui"
)

type optimizationPassSet map[optimizationPass]bool

func optimizationPipeline(mode OptimizationMode) optimizationPassSet {
	passes := make(optimizationPassSet)
	switch mode {
	case OptimizationControlState:
		passes[passControlReachability] = true
		passes[passTrimProgramUI] = true
		passes[passTrimStackUI] = true
	}
	return passes
}

type runtimePlan struct {
	optimization OptimizationMode
	profile      profile.Machine
	passes       optimizationPassSet
	usedPhases   map[int]bool
	reachable    [runtimeProgWords][runtimeStackSize + 1]bool
	maxPC        int
	maxSP        int
}

func instructionPhaseRange(name string) (start, end int, ok bool) {
	switch name {
	case "PSH":
		return phasePSHStart, phasePSHEnd, true
	case "ADD":
		return phaseADDStart, phaseADDEnd, true
	case "SUB":
		return phaseSUBStart, phaseSUBEnd, true
	case "DUP":
		return phaseDUPStart, phaseDUPEnd, true
	case "POP":
		return phasePOPStart, phasePOPEnd, true
	case "SWP":
		return phaseSWPStart, phaseSWPEnd, true
	case "OVR":
		return phaseOVRStart, phaseOVREnd, true
	case "INC":
		return phaseINCStart, phaseINCEnd, true
	case "DEC":
		return phaseDECStart, phaseDECEnd, true
	case "CLL":
		return phaseCLLStart, phaseCLLEnd, true
	case "RET":
		return phaseRETStart, phaseRETEnd, true
	case "AND":
		return phaseANDStart, phaseANDEnd, true
	case "XOR":
		return phaseXORStart, phaseXOREnd, true
	case "OR":
		return phaseORStart, phaseOREnd, true
	case "NOT":
		return phaseNOTStart, phaseNOTEnd, true
	case "SHL":
		return phaseSHLStart, phaseSHLEnd, true
	case "SHR":
		return phaseSHRStart, phaseSHREnd, true
	case "JMP":
		return phaseJMPStart, phaseJMPEnd, true
	case "JNZ":
		return phaseJNZStart, phaseJNZEnd, true
	case "SAY":
		return phaseSAYStart, phaseSAYEnd, true
	default:
		return 0, 0, false
	}
}

func collectUsedPhases(words []decodedWord, prof profile.Machine) map[int]bool {
	used := map[int]bool{phaseDispatch: true}
	for pc := 0; pc < prof.ProgramWords; pc++ {
		start, end, ok := instructionPhaseRange(words[pc].Name)
		if !ok {
			continue
		}
		for ph := start; ph <= end; ph++ {
			used[ph] = true
		}
	}
	return used
}

func buildRuntimePlan(words []decodedWord, opts Options) runtimePlan {
	opts = normalizedOptions(opts)
	passes := optimizationPipeline(opts.Optimization)
	plan := runtimePlan{
		optimization: opts.Optimization,
		profile:      opts.Profile,
		passes:       passes,
		usedPhases:   collectUsedPhases(words, opts.Profile),
		maxPC:        opts.Profile.ProgramWords - 1,
		maxSP:        opts.Profile.StackWords,
	}
	if !plan.passes[passControlReachability] {
		return plan
	}

	reach := analyzeReachableControlStates(words, opts.Profile)
	plan.reachable = reach
	plan.maxPC = 0
	plan.maxSP = 0
	for pc := 0; pc < opts.Profile.ProgramWords; pc++ {
		for sp := 0; sp <= opts.Profile.StackWords; sp++ {
			if !reach[pc][sp] {
				continue
			}
			if pc > plan.maxPC {
				plan.maxPC = pc
			}
			if sp > plan.maxSP {
				plan.maxSP = sp
			}
		}
	}
	return plan
}

func (p runtimePlan) active(pc, sp int) bool {
	if pc < 0 || pc >= p.profile.ProgramWords || sp < 0 || sp > p.profile.StackWords {
		return false
	}
	if !p.passes[passControlReachability] {
		return true
	}
	return p.reachable[pc][sp]
}

func (p runtimePlan) visibleStackCells() int {
	if !p.passes[passTrimStackUI] {
		return p.profile.StackWords
	}
	cells := p.maxSP
	if cells < 1 {
		cells = 1
	}
	if cells > p.profile.StackWords {
		cells = p.profile.StackWords
	}
	return cells
}

func (p runtimePlan) visibleProgramWords() int {
	if !p.passes[passTrimProgramUI] {
		return p.profile.ProgramWords
	}
	words := p.maxPC + 1
	if words < 1 {
		words = 1
	}
	if words > p.profile.ProgramWords {
		words = p.profile.ProgramWords
	}
	return words
}

func (p runtimePlan) phaseActive(ph int) bool {
	return p.usedPhases[ph]
}

func analyzeReachableControlStates(
	words []decodedWord,
	prof profile.Machine,
) [runtimeProgWords][runtimeStackSize + 1]bool {
	var reach [runtimeProgWords][runtimeStackSize + 1]bool
	type state struct {
		pc int
		sp int
	}

	q := make([]state, 0, prof.ProgramWords*(prof.StackWords+1))
	enqueue := func(pc, sp int) {
		if pc < 0 || pc >= prof.ProgramWords || sp < 0 || sp > prof.StackWords {
			return
		}
		if reach[pc][sp] {
			return
		}
		reach[pc][sp] = true
		q = append(q, state{pc: pc, sp: sp})
	}
	enqueue(0, 0)

	for i := 0; i < len(q); i++ {
		s := q[i]
		pc := s.pc
		sp := s.sp
		nextPC := pc + 1
		word := words[pc]

		switch word.Name {
		case "HLT":
			// terminal state
		case "PSH":
			if sp < prof.StackWords {
				enqueue(nextPC, sp+1)
			}
		case "POP":
			if sp >= 1 {
				enqueue(nextPC, sp-1)
			}
		case "ADD", "SUB", "AND", "XOR", "OR":
			if sp >= 2 {
				enqueue(nextPC, sp-1)
			}
		case "DUP":
			if sp >= 1 && sp < prof.StackWords {
				enqueue(nextPC, sp+1)
			}
		case "NOT", "SHL", "SHR", "SAY":
			if sp >= 1 {
				enqueue(nextPC, sp)
			}
		case "INC", "DEC":
			if sp >= 1 {
				enqueue(nextPC, sp)
			}
		case "SWP":
			if sp >= 2 {
				enqueue(nextPC, sp)
			}
		case "OVR":
			if sp >= 2 && sp < prof.StackWords {
				enqueue(nextPC, sp+1)
			}
		case "CLL":
			if sp < prof.StackWords {
				enqueue(int(word.Operand), sp+1)
			}
		case "RET":
			if sp >= 1 {
				// Return target is value-dependent; keep all in-range PCs.
				for targetPC := 0; targetPC < prof.ProgramWords; targetPC++ {
					enqueue(targetPC, sp-1)
				}
			}
		case "JMP":
			enqueue(int(word.Operand), sp)
		case "JNZ":
			if sp >= 1 {
				// Control-state analysis keeps both branches.
				enqueue(int(word.Operand), sp)
				enqueue(nextPC, sp)
			}
		}
	}

	return reach
}

func validateRuntimeSpec(spec *lvdl.Spec, prof profile.Machine) error {
	if spec.Machine.WordBits != prof.WordBits {
		return fmt.Errorf("profile %s requires (word %d), got %d", prof.Name, prof.WordBits, spec.Machine.WordBits)
	}
	if spec.ISA.Encoding.OpcodeBits != prof.OpcodeBits || spec.ISA.Encoding.OperandBits != prof.OperandBits {
		return fmt.Errorf(
			"profile %s requires (encoding (opcode-bits %d) (operand-bits %d)), got %d/%d",
			prof.Name, prof.OpcodeBits, prof.OperandBits,
			spec.ISA.Encoding.OpcodeBits, spec.ISA.Encoding.OperandBits,
		)
	}
	if len(spec.Machine.Stacks) == 0 {
		return fmt.Errorf("runtime8 requires one stack declaration")
	}
	stk := spec.Machine.Stacks[0]
	if stk.Words != prof.StackWords || stk.Bits != prof.StackBits {
		return fmt.Errorf(
			"profile %s requires first stack to be (stack %s %d %d), got (%d %d)",
			prof.Name, stk.Name, prof.StackWords, prof.StackBits, stk.Words, stk.Bits,
		)
	}

	required := []string{"HLT", "PSH", "POP", "ADD", "SUB", "AND", "XOR", "OR", "DUP", "SWP", "OVR", "INC", "DEC", "CLL", "RET", "NOT", "SHL", "SHR", "JMP", "JNZ", "SAY"}
	for _, name := range required {
		if _, ok := spec.ISA.ByName[name]; !ok {
			return fmt.Errorf("runtime8 requires ISA instruction %q", name)
		}
	}

	pcBits := -1
	spBits := -1
	for _, r := range spec.Machine.Regs {
		switch strings.ToUpper(r.Name) {
		case "PC":
			pcBits = r.Bits
		case "SP":
			spBits = r.Bits
		}
	}
	if pcBits != prof.PCBits {
		return fmt.Errorf("profile %s requires (reg PC %d), got %d", prof.Name, prof.PCBits, pcBits)
	}
	if spBits != prof.SPBits {
		return fmt.Errorf("profile %s requires (reg SP %d), got %d", prof.Name, prof.SPBits, spBits)
	}

	return nil
}

func normalizeProgram(program []uint16, prof profile.Machine) ([runtimeProgWords]uint16, error) {
	var out [runtimeProgWords]uint16
	if prof.ProgramWords > runtimeProgWords || prof.StackWords > runtimeStackSize || prof.StackBits != runtimeWordBits {
		return out, fmt.Errorf(
			"profile %s exceeds runtime8 core limits (max program=%d stack=%d bits=%d)",
			prof.Name, runtimeProgWords, runtimeStackSize, runtimeWordBits,
		)
	}
	if len(program) > prof.ProgramWords {
		return out, fmt.Errorf(
			"runtime8 supports up to %d words, got %d",
			prof.ProgramWords, len(program),
		)
	}
	copy(out[:], program)
	return out, nil
}

func decodeProgramWords(
	spec *lvdl.Spec,
	program [runtimeProgWords]uint16,
	prof profile.Machine,
) ([]decodedWord, error) {
	out := make([]decodedWord, runtimeProgWords)
	for pc := 0; pc < runtimeProgWords; pc++ {
		if pc >= prof.ProgramWords {
			out[pc] = decodedWord{Name: "HLT", Text: "HLT"}
			continue
		}
		w := program[pc]
		op := uint8(w >> 8)
		imm := uint8(w & 0x00FF)

		ins, ok := spec.ISA.ByOpcode[op]
		if !ok {
			return nil, fmt.Errorf("unknown opcode 0x%02X at pc %d", op, pc)
		}
		name := strings.ToUpper(ins.Name)
		switch name {
		case "HLT", "PSH", "POP", "ADD", "SUB", "AND", "XOR", "OR", "DUP", "SWP", "OVR", "INC", "DEC", "CLL", "RET", "NOT", "SHL", "SHR", "JMP", "JNZ", "SAY":
			// supported in this branch
		default:
			return nil, fmt.Errorf(
				"unsupported instruction %q at pc %d (runtime8 supports HLT/PSH/POP/ADD/SUB/AND/XOR/OR/DUP/SWP/OVR/INC/DEC/CLL/RET/NOT/SHL/SHR/JMP/JNZ/SAY)",
				name, pc,
			)
		}
		if (name == "JMP" || name == "JNZ" || name == "CLL") && int(imm) >= prof.ProgramWords {
			return nil, fmt.Errorf(
				"%s target out of range at pc %d: %d (max %d)",
				strings.ToLower(name), pc, imm, prof.ProgramWords-1,
			)
		}
		if name != "HLT" && name != "JMP" && name != "CLL" && name != "RET" && pc == prof.ProgramWords-1 {
			return nil, fmt.Errorf("non-HLT instruction at pc %d has no in-range fallthrough", pc)
		}

		text := name
		if name == "PSH" {
			text = fmt.Sprintf("PSH %d", imm)
		}
		if name == "JMP" {
			text = fmt.Sprintf("JMP %d", imm)
		}
		if name == "JNZ" {
			text = fmt.Sprintf("JNZ %d", imm)
		}
		if name == "CLL" {
			text = fmt.Sprintf("CLL %d", imm)
		}

		out[pc] = decodedWord{Opcode: op, Operand: imm, Name: name, Text: text}
	}
	return out, nil
}

func has(id string) string {
	return fmt.Sprintf(":has(#%s:checked)", id)
}

func notHas(id string) string {
	return fmt.Sprintf(":not(:has(#%s:checked))", id)
}

func pcID(pc int) string {
	return fmt.Sprintf("p%d", pc)
}

func spID(sp int) string {
	return fmt.Sprintf("s%d", sp)
}

func ospID(sp int) string {
	return fmt.Sprintf("os%d", sp)
}

func opcID(pc int) string {
	return fmt.Sprintf("op%d", pc)
}

func phaseID(ph int) string {
	return fmt.Sprintf("h%d", ph)
}

func stkBitID(cell, bit, v int) string {
	return fmt.Sprintf("k%d%d%d", cell, bit, v)
}

func sumBitID(bit, v int) string {
	return fmt.Sprintf("u%d%d", bit, v)
}

func carBitID(bit, v int) string {
	return fmt.Sprintf("c%d%d", bit, v)
}

func outBitID(bit, v int) string {
	return fmt.Sprintf("o%d%d", bit, v)
}

func outValidID(v int) string {
	return fmt.Sprintf("ov%d", v)
}

func outValueSel(v int) string {
	var b strings.Builder
	for bit := 0; bit < runtimeWordBits; bit++ {
		bitVal := (v >> bit) & 1
		b.WriteString(has(outBitID(bit, bitVal)))
	}
	return b.String()
}

func stackValueSel(cell, v int) string {
	var b strings.Builder
	for bit := 0; bit < runtimeWordBits; bit++ {
		bitVal := (v >> bit) & 1
		b.WriteString(has(stkBitID(cell, bit, bitVal)))
	}
	return b.String()
}

func minifyCSS(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = cssSpaceAroundTokenRe.ReplaceAllString(s, `$1`)
	s = cssMultiSpaceRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, ";}", "}")
	return strings.TrimSpace(s)
}

func writeStateRadiosHTML(w io.Writer, plan runtimePlan) {
	// Phase register
	for ph := phaseDispatch; ph <= runtimeMaxPhase; ph++ {
		if !plan.phaseActive(ph) {
			continue
		}
		checked := ""
		if ph == phaseDispatch {
			checked = " checked"
		}
		fmt.Fprintf(
			w,
			"<input type=\"radio\" name=\"phase\" id=\"%s\"%s>\n",
			phaseID(ph), checked,
		)
	}

	// PC one-hot
	for pc := 0; pc < plan.profile.ProgramWords; pc++ {
		checked := ""
		if pc == 0 {
			checked = " checked"
		}
		fmt.Fprintf(
			w,
			"<input type=\"radio\" name=\"pc\" id=\"%s\"%s>\n",
			pcID(pc), checked,
		)
	}

	// SP one-hot
	for sp := 0; sp <= plan.profile.StackWords; sp++ {
		checked := ""
		if sp == 0 {
			checked = " checked"
		}
		fmt.Fprintf(
			w,
			"<input type=\"radio\" name=\"sp\" id=\"%s\"%s>\n",
			spID(sp), checked,
		)
	}

	// Operation-start PC latch (binds instruction micro-steps to the
	// PC that was active at dispatch time).
	for pc := 0; pc < plan.profile.ProgramWords; pc++ {
		checked := ""
		if pc == 0 {
			checked = " checked"
		}
		fmt.Fprintf(
			w,
			"<input type=\"radio\" name=\"opc\" id=\"%s\"%s>\n",
			opcID(pc), checked,
		)
	}

	// Operation-start SP latch (binds instruction micro-steps to the
	// SP that was active at dispatch time).
	for sp := 0; sp <= plan.profile.StackWords; sp++ {
		checked := ""
		if sp == 0 {
			checked = " checked"
		}
		fmt.Fprintf(
			w,
			"<input type=\"radio\" name=\"osp\" id=\"%s\"%s>\n",
			ospID(sp), checked,
		)
	}

	// Stack bit radios
	for cell := 0; cell < plan.profile.StackWords; cell++ {
		for bit := 0; bit < runtimeWordBits; bit++ {
			group := fmt.Sprintf("stk%d_b%d", cell, bit)
			fmt.Fprintf(w, "<input type=\"radio\" name=\"%s\" id=\"%s\" checked>\n", group, stkBitID(cell, bit, 0))
			fmt.Fprintf(w, "<input type=\"radio\" name=\"%s\" id=\"%s\">\n", group, stkBitID(cell, bit, 1))
		}
	}

	// ALU staging bits
	for bit := 0; bit < runtimeWordBits; bit++ {
		group := fmt.Sprintf("sum_b%d", bit)
		fmt.Fprintf(w, "<input type=\"radio\" name=\"%s\" id=\"%s\" checked>\n", group, sumBitID(bit, 0))
		fmt.Fprintf(w, "<input type=\"radio\" name=\"%s\" id=\"%s\">\n", group, sumBitID(bit, 1))
	}
	for bit := 0; bit <= runtimeWordBits; bit++ {
		group := fmt.Sprintf("car%d", bit)
		fmt.Fprintf(w, "<input type=\"radio\" name=\"%s\" id=\"%s\" checked>\n", group, carBitID(bit, 0))
		fmt.Fprintf(w, "<input type=\"radio\" name=\"%s\" id=\"%s\">\n", group, carBitID(bit, 1))
	}

	// Output bits + valid flag
	for bit := 0; bit < runtimeWordBits; bit++ {
		group := fmt.Sprintf("out_b%d", bit)
		fmt.Fprintf(w, "<input type=\"radio\" name=\"%s\" id=\"%s\" checked>\n", group, outBitID(bit, 0))
		fmt.Fprintf(w, "<input type=\"radio\" name=\"%s\" id=\"%s\">\n", group, outBitID(bit, 1))
	}
	fmt.Fprintf(w, "<input type=\"radio\" name=\"out_valid\" id=\"%s\" checked>\n", outValidID(0))
	fmt.Fprintf(w, "<input type=\"radio\" name=\"out_valid\" id=\"%s\">\n", outValidID(1))
}

func genRuntimeCSS(words []decodedWord, plan runtimePlan) string {
	var b strings.Builder

	darkVars := `  --bg:#000;--bg-panel:#111;
  --text:#d4d4d4;--text-muted:#888;
  --border:#2a2a2a;--border-hover:#555;
  --bg-active:#1a1a1a;--bg-top:#1a1a1a;
  --highlight-bg:#0a1428;
  --badge-css-bg:#0a1428;`

	b.WriteString(`
*{box-sizing:border-box}
:root{
  font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,"Liberation Mono",monospace;
  --bg:#fff;--bg-panel:#fff;
  --text:#111;--text-muted:#666;
  --border:#ddd;--border-hover:#666;
  --bg-active:#eee;--bg-top:#f8f8f8;
  --highlight-bg:#eef3ff;
  --badge-css-bg:#e8f0fe;
}
`)
	b.WriteString(themeDarkCSS(darkVars))
	b.WriteString(`
body{margin:0;padding:16px;background:var(--bg);color:var(--text)}
input[type="radio"]{position:absolute;opacity:0;pointer-events:none;width:0;height:0;overflow:hidden}
.app{max-width:1200px;margin:0 auto;display:grid;grid-template-columns:380px 1fr;gap:10px}
.title{grid-column:1 / -1;display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.title h1{margin:0;font-size:18px}
.badge-css{display:inline-block;font-size:10px;padding:1px 6px;border-radius:4px;background:var(--badge-css-bg);color:#2965f1;border:1px solid #2965f1;font-weight:700}
.badge-js{display:inline-block;font-size:10px;padding:1px 6px;border-radius:4px;background:#2b1a00;color:#ffb020;border:1px solid #ffb020;font-weight:700}
.status{font-size:11px;padding:2px 8px;border-radius:999px;border:1px solid var(--border)}
.status-halted{display:none;border-color:#999;color:#999}
.clock{display:flex;align-items:center;gap:6px;flex-wrap:wrap}
.clock button{border:1px solid var(--border);background:var(--bg-panel);color:var(--text);font:inherit;padding:2px 8px;border-radius:999px;cursor:pointer}
.clock button:hover{border-color:var(--border-hover)}
.clock button.speed-active{background:var(--highlight-bg);border-color:#2965f1}
.panel{border:1px solid var(--border);border-radius:14px;padding:10px;background:var(--bg-panel)}
.panel h3{margin:0 0 8px 0;font-size:13px}
.kv{display:grid;grid-template-columns:1fr 1fr;gap:8px}
.kv .box{border:1px solid var(--border);border-radius:10px;padding:8px}
.kv .label{font-size:11px;font-weight:700;color:var(--text-muted)}
.kv .value{font-size:22px;font-weight:700}
.kv .value span{display:none}
.write-bus{position:relative;min-height:44px;border:1px solid var(--border);border-radius:10px;overflow:hidden}
.wz{position:absolute;inset:0;display:flex;visibility:hidden;pointer-events:none}
.wz-next{z-index:30}
.wz-set{z-index:20}
.wz-reset{z-index:15}
.wz-dispatch{z-index:10}
.wz label{display:flex;align-items:center;justify-content:center;width:100%;height:100%;cursor:pointer;font-size:12px;font-weight:700;color:var(--text)}
.wz label:hover{background:var(--highlight-bg)}
.bus-idle{display:none;position:absolute;inset:0;align-items:center;justify-content:center;font-size:12px;color:var(--text-muted)}
.stack{display:grid;gap:4px;max-height:260px;overflow:auto}
.stk-row{border:1px solid var(--border);border-radius:8px;padding:4px 8px;font-size:12px}
.stk-row .bit span{display:none}
.stk-dec{display:inline-block;min-width:3ch;text-align:right;font-weight:700;color:var(--text-muted);margin-left:8px}
.stk-dec::before{content:"0"}
.program{max-height:520px;overflow:auto}
.pl{font-size:12px;padding:2px 8px;border-left:3px solid transparent;white-space:pre}
.ov{font-size:20px;font-weight:700}
.ov::before{content:"-";color:var(--text-muted)}
@media(max-width:980px){.app{grid-template-columns:1fr}}
`)
	b.WriteString(themeToggleCSS())

	// PC and SP displays
	for pc := 0; pc < plan.visibleProgramWords(); pc++ {
		fmt.Fprintf(&b, "body%s .pd .p%d{display:inline}\n", has(pcID(pc)), pc)
	}
	for sp := 0; sp <= runtimeStackSize; sp++ {
		fmt.Fprintf(&b, "body%s .sd .s%d{display:inline}\n", has(spID(sp)), sp)
	}

	// Current instruction label and highlight
	for pc := 0; pc < plan.visibleProgramWords(); pc++ {
		fmt.Fprintf(&b, "body%s .id .i%d{display:inline}\n", has(pcID(pc)), pc)
		fmt.Fprintf(&b, "body%s .pl[data-p=\"%d\"]{border-left-color:#2965f1;background:var(--highlight-bg)}\n", has(pcID(pc)), pc)
	}

	// Stack bit displays
	visibleCells := plan.visibleStackCells()
	for cell := 0; cell < visibleCells; cell++ {
		for bit := 0; bit < runtimeWordBits; bit++ {
			for v := 0; v <= 1; v++ {
				fmt.Fprintf(
					&b,
					"body%s .k%d%d%d{display:inline}\n",
					has(stkBitID(cell, bit, v)), cell, bit, v,
				)
			}
		}
	}
	for cell := 0; cell < visibleCells; cell++ {
		for v := 0; v < 256; v++ {
			sel := "body" + stackValueSel(cell, v)
			fmt.Fprintf(
				&b,
				"%s .d%d::before{content:\"%d\"}\n",
				sel, cell, v,
			)
		}
	}

	// Output display
	for v := 0; v < 256; v++ {
		sel := "body" + has(outValidID(1)) + outValueSel(v)
		fmt.Fprintf(&b, "%s .ov::before{content:\"%d\";color:var(--text)}\n", sel, v)
	}

	// Halt state
	for pc := 0; pc < plan.visibleProgramWords(); pc++ {
		if words[pc].Name != "HLT" {
			continue
		}
		sel := "body" + has(phaseID(phaseDispatch)) + has(pcID(pc))
		fmt.Fprintf(&b, "%s .status-running{display:none}\n", sel)
		fmt.Fprintf(&b, "%s .status-halted{display:inline-block}\n", sel)
		fmt.Fprintf(&b, "%s .bus-idle{display:flex}\n", sel)
	}

	// Dispatch (phase 0)
	for pc := 0; pc < runtimeProgWords; pc++ {
		switch words[pc].Name {
		case "PSH":
			for sp := 0; sp < runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "ADD":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "SUB":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "AND":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "XOR":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "OR":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "DUP":
			for sp := 1; sp < runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "POP":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "SWP":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "OVR":
			for sp := 2; sp < runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "INC":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "DEC":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "CLL":
			for sp := 0; sp < runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "RET":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "NOT":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "SHL":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "SHR":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "JMP":
			for sp := 0; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "JNZ":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		case "SAY":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				ospSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					notHas(ospID(sp))
				opcSetSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					notHas(opcID(pc))
				nextSel := "body" +
					has(phaseID(phaseDispatch)) +
					has(pcID(pc)) +
					has(spID(sp)) +
					has(ospID(sp)) +
					has(opcID(pc))
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-osp-set{visibility:visible;pointer-events:auto}\n", ospSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-opc-set{visibility:visible;pointer-events:auto}\n", opcSetSel, pc, sp)
				fmt.Fprintf(&b, "%s .wz-dispatch-p%d-s%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp)
			}
		}
	}

	genPSHCSS(&b, words, plan)
	genADDCSS(&b, words, plan)
	genDUPCSS(&b, words, plan)
	genPOPCSS(&b, words, plan)
	genSWPCSS(&b, words, plan)
	genOVRCSS(&b, words, plan)
	genINCCSS(&b, words, plan)
	genDECCSS(&b, words, plan)
	genSUBCSS(&b, words, plan)
	genANDCSS(&b, words, plan)
	genXORCSS(&b, words, plan)
	genORCSS(&b, words, plan)
	genNOTCSS(&b, words, plan)
	genSHLCSS(&b, words, plan)
	genSHRCSS(&b, words, plan)
	genCLLCSS(&b, words, plan)
	genRETCSS(&b, words, plan)
	genJMPCSS(&b, words, plan)
	genJNZCSS(&b, words, plan)
	genSAYCSS(&b, words, plan)

	return minifyCSS(b.String())
}

func genWriteZonesHTML(w io.Writer, words []decodedWord, plan runtimePlan) {
	io.WriteString(w, "<div class=\"bus-idle\">HLT</div>")

	for pc := 0; pc < runtimeProgWords; pc++ {
		switch words[pc].Name {
		case "PSH":
			for sp := 0; sp < runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d p</label></div>",
					pc, sp, phaseID(phasePSHStart),
				)
			}
		case "ADD":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d a</label></div>",
					pc, sp, phaseID(phaseADDStart),
				)
			}
		case "SUB":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d b</label></div>",
					pc, sp, phaseID(phaseSUBStart),
				)
			}
		case "AND":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d n</label></div>",
					pc, sp, phaseID(phaseANDStart),
				)
			}
		case "XOR":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d x</label></div>",
					pc, sp, phaseID(phaseXORStart),
				)
			}
		case "OR":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d o</label></div>",
					pc, sp, phaseID(phaseORStart),
				)
			}
		case "DUP":
			for sp := 1; sp < runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d u</label></div>",
					pc, sp, phaseID(phaseDUPStart),
				)
			}
		case "POP":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d p</label></div>",
					pc, sp, phaseID(phasePOPStart),
				)
			}
		case "SWP":
			for sp := 2; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d w</label></div>",
					pc, sp, phaseID(phaseSWPStart),
				)
			}
		case "OVR":
			for sp := 2; sp < runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d v</label></div>",
					pc, sp, phaseID(phaseOVRStart),
				)
			}
		case "INC":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d i</label></div>",
					pc, sp, phaseID(phaseINCStart),
				)
			}
		case "DEC":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d e</label></div>",
					pc, sp, phaseID(phaseDECStart),
				)
			}
		case "CLL":
			for sp := 0; sp < runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d c</label></div>",
					pc, sp, phaseID(phaseCLLStart),
				)
			}
		case "RET":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d r</label></div>",
					pc, sp, phaseID(phaseRETStart),
				)
			}
		case "NOT":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d t</label></div>",
					pc, sp, phaseID(phaseNOTStart),
				)
			}
		case "SHL":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d l</label></div>",
					pc, sp, phaseID(phaseSHLStart),
				)
			}
		case "SHR":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d r</label></div>",
					pc, sp, phaseID(phaseSHRStart),
				)
			}
		case "JMP":
			for sp := 0; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d j</label></div>",
					pc, sp, phaseID(phaseJMPStart),
				)
			}
		case "JNZ":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d z</label></div>",
					pc, sp, phaseID(phaseJNZStart),
				)
			}
		case "SAY":
			for sp := 1; sp <= runtimeStackSize; sp++ {
				if !plan.active(pc, sp) {
					continue
				}
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-osp-set\"><label for=\"%s\">os=%d</label></div>",
					pc, sp, ospID(sp), sp,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-set wz-dispatch-p%d-s%d-opc-set\"><label for=\"%s\">op=%d</label></div>",
					pc, sp, opcID(pc), pc,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-dispatch wz-next wz-dispatch-p%d-s%d-next\"><label for=\"%s\">d s</label></div>",
					pc, sp, phaseID(phaseSAYStart),
				)
			}
		}
	}

	// PSH labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "PSH" {
			continue
		}
		imm := int(words[pc].Operand)
		for sp := 0; sp < runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phasePSHStart + bit
				bitVal := (imm >> bit) & 1
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-psh-p%d-s%d-b%d-set%d\"><label for=\"%s\">b%d=%d</label></div>",
					pc, sp, bit, bitVal, stkBitID(sp, bit, bitVal), bit, bitVal,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-psh-p%d-s%d-b%d-next\"><label for=\"%s\">></label></div>",
					pc, sp, bit, phaseID(ph+1),
				)
			}
			fmt.Fprintf(w, "<div class=\"wz wz-psh-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp+1), sp+1)
			fmt.Fprintf(w, "<div class=\"wz wz-psh-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phasePSHStart+9))
			fmt.Fprintf(w, "<div class=\"wz wz-psh-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-psh-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phasePSHEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-psh-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// ADD labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "ADD" {
			continue
		}
		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			fmt.Fprintf(w, "<div class=\"wz wz-add-p%d-s%d-car0-set\"><label for=\"%s\">c0=0</label></div>", pc, sp, carBitID(0, 0))
			fmt.Fprintf(w, "<div class=\"wz wz-add-p%d-s%d-car0-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseADDStart+1))

			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseADDStart + 1 + bit*2
				carPhase := sumPhase + 1

				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						for cv := 0; cv <= 1; cv++ {
							s := sumBit(av, bv, cv)
							cnext := carryBit(av, bv, cv)
							fmt.Fprintf(
								w,
								"<div class=\"wz wz-add-p%d-s%d-sum%d-set%d-a%d-b%d-c%d\"><label for=\"%s\">s%d=%d</label></div>",
								pc, sp, bit, s, av, bv, cv, sumBitID(bit, s), bit, s,
							)
							fmt.Fprintf(
								w,
								"<div class=\"wz wz-add-p%d-s%d-sum%d-next-a%d-b%d-c%d\"><label for=\"%s\">></label></div>",
								pc, sp, bit, av, bv, cv, phaseID(carPhase),
							)
							fmt.Fprintf(
								w,
								"<div class=\"wz wz-add-p%d-s%d-car%d-set%d-a%d-b%d-c%d\"><label for=\"%s\">c%d=%d</label></div>",
								pc, sp, bit+1, cnext, av, bv, cv, carBitID(bit+1, cnext), bit+1, cnext,
							)
							fmt.Fprintf(
								w,
								"<div class=\"wz wz-add-p%d-s%d-car%d-next-a%d-b%d-c%d\"><label for=\"%s\">></label></div>",
								pc, sp, bit+1, av, bv, cv, phaseID(carPhase+1),
							)
						}
					}
				}
			}

			wbStart := phaseADDStart + 17
			dest := sp - 2
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-add-p%d-s%d-wb%d-set%d-sum%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dest, bit, sv), dest, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-add-p%d-s%d-wb%d-next-sum%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(wbStart+bit+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-add-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp-1), sp-1)
			fmt.Fprintf(w, "<div class=\"wz wz-add-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseADDStart+26))
			fmt.Fprintf(w, "<div class=\"wz wz-add-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-add-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseADDEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-add-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// DUP labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "DUP" {
			continue
		}
		for sp := 1; sp < runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			dst := sp
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseDUPStart + bit
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-dup-p%d-s%d-b%d-set%d-s%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dst, bit, sv), dst, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-dup-p%d-s%d-b%d-next-s%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(ph+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-dup-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp+1), sp+1)
			fmt.Fprintf(w, "<div class=\"wz wz-dup-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseDUPStart+9))
			fmt.Fprintf(w, "<div class=\"wz wz-dup-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-dup-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseDUPEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-dup-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// POP labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "POP" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			fmt.Fprintf(w, "<div class=\"wz wz-pop-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp-1), sp-1)
			fmt.Fprintf(w, "<div class=\"wz wz-pop-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phasePOPStart+1))
			fmt.Fprintf(w, "<div class=\"wz wz-pop-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-pop-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phasePOPEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-pop-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// SWP labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SWP" {
			continue
		}
		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			top := sp - 1
			under := sp - 2
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseSWPStart + bit
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-swp-p%d-s%d-u%d-set%d-s%d\"><label for=\"%s\">u%d=%d</label></div>",
						pc, sp, bit, sv, sv, sumBitID(bit, sv), bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-swp-p%d-s%d-u%d-next-s%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(ph+1),
					)
				}
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseSWPStart + 8 + bit
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-swp-p%d-s%d-top%d-set%d-s%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(top, bit, sv), top, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-swp-p%d-s%d-top%d-next-s%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(ph+1),
					)
				}
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseSWPStart + 16 + bit
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-swp-p%d-s%d-low%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(under, bit, sv), under, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-swp-p%d-s%d-low%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(ph+1),
					)
				}
			}
			fmt.Fprintf(w, "<div class=\"wz wz-swp-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-swp-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseSWPEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-swp-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// OVR labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "OVR" {
			continue
		}
		for sp := 2; sp < runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			dst := sp
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseOVRStart + bit
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-ovr-p%d-s%d-b%d-set%d-s%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dst, bit, sv), dst, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-ovr-p%d-s%d-b%d-next-s%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(ph+1),
					)
				}
			}
			fmt.Fprintf(w, "<div class=\"wz wz-ovr-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp+1), sp+1)
			fmt.Fprintf(w, "<div class=\"wz wz-ovr-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseOVRStart+9))
			fmt.Fprintf(w, "<div class=\"wz wz-ovr-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-ovr-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseOVREnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-ovr-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// INC labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "INC" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			top := sp - 1
			fmt.Fprintf(w, "<div class=\"wz wz-inc-p%d-s%d-c0-set\"><label for=\"%s\">c0=1</label></div>", pc, sp, carBitID(0, 1))
			fmt.Fprintf(w, "<div class=\"wz wz-inc-p%d-s%d-c0-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseINCStart+1))
			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseINCStart + 1 + bit*2
				carPhase := sumPhase + 1
				for av := 0; av <= 1; av++ {
					for cv := 0; cv <= 1; cv++ {
						s := sumBit(av, 0, cv)
						cnext := carryBit(av, 0, cv)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-inc-p%d-s%d-u%d-set%d-a%d-c%d\"><label for=\"%s\">u%d=%d</label></div>",
							pc, sp, bit, s, av, cv, sumBitID(bit, s), bit, s,
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-inc-p%d-s%d-u%d-next-a%d-c%d\"><label for=\"%s\">></label></div>",
							pc, sp, bit, av, cv, phaseID(carPhase),
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-inc-p%d-s%d-c%d-set%d-a%d-c%d\"><label for=\"%s\">c%d=%d</label></div>",
							pc, sp, bit+1, cnext, av, cv, carBitID(bit+1, cnext), bit+1, cnext,
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-inc-p%d-s%d-c%d-next-a%d-c%d\"><label for=\"%s\">></label></div>",
							pc, sp, bit+1, av, cv, phaseID(carPhase+1),
						)
					}
				}
			}
			wbStart := phaseINCStart + 17
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := wbStart + bit
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-inc-p%d-s%d-wb%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(top, bit, sv), top, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-inc-p%d-s%d-wb%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(ph+1),
					)
				}
			}
			fmt.Fprintf(w, "<div class=\"wz wz-inc-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-inc-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseINCEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-inc-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// DEC labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "DEC" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			top := sp - 1
			fmt.Fprintf(w, "<div class=\"wz wz-dec-p%d-s%d-c0-set\"><label for=\"%s\">c0=1</label></div>", pc, sp, carBitID(0, 1))
			fmt.Fprintf(w, "<div class=\"wz wz-dec-p%d-s%d-c0-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseDECStart+1))
			for bit := 0; bit < runtimeWordBits; bit++ {
				diffPhase := phaseDECStart + 1 + bit*2
				borrowPhase := diffPhase + 1
				for av := 0; av <= 1; av++ {
					for bin := 0; bin <= 1; bin++ {
						d := diffBit(av, 0, bin)
						bout := borrowBit(av, 0, bin)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-dec-p%d-s%d-u%d-set%d-a%d-c%d\"><label for=\"%s\">u%d=%d</label></div>",
							pc, sp, bit, d, av, bin, sumBitID(bit, d), bit, d,
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-dec-p%d-s%d-u%d-next-a%d-c%d\"><label for=\"%s\">></label></div>",
							pc, sp, bit, av, bin, phaseID(borrowPhase),
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-dec-p%d-s%d-c%d-set%d-a%d-c%d\"><label for=\"%s\">c%d=%d</label></div>",
							pc, sp, bit+1, bout, av, bin, carBitID(bit+1, bout), bit+1, bout,
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-dec-p%d-s%d-c%d-next-a%d-c%d\"><label for=\"%s\">></label></div>",
							pc, sp, bit+1, av, bin, phaseID(borrowPhase+1),
						)
					}
				}
			}
			wbStart := phaseDECStart + 17
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := wbStart + bit
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-dec-p%d-s%d-wb%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(top, bit, sv), top, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-dec-p%d-s%d-wb%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(ph+1),
					)
				}
			}
			fmt.Fprintf(w, "<div class=\"wz wz-dec-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-dec-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseDECEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-dec-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// CLL labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "CLL" {
			continue
		}
		targetPC := int(words[pc].Operand)
		returnPC := pc + 1
		for sp := 0; sp < runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseCLLStart + bit
				bitVal := (returnPC >> bit) & 1
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-cll-p%d-s%d-b%d-set%d\"><label for=\"%s\">b%d=%d</label></div>",
					pc, sp, bit, bitVal, stkBitID(sp, bit, bitVal), bit, bitVal,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-cll-p%d-s%d-b%d-next\"><label for=\"%s\">></label></div>",
					pc, sp, bit, phaseID(ph+1),
				)
			}
			fmt.Fprintf(w, "<div class=\"wz wz-cll-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp+1), sp+1)
			fmt.Fprintf(w, "<div class=\"wz wz-cll-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseCLLStart+9))
			fmt.Fprintf(w, "<div class=\"wz wz-cll-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(targetPC), targetPC)
			fmt.Fprintf(w, "<div class=\"wz wz-cll-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseCLLEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-cll-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// RET labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "RET" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for v := 0; v < plan.profile.ProgramWords; v++ {
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-ret-p%d-s%d-pc-set-v%d\"><label for=\"%s\">pc=%d</label></div>",
					pc, sp, v, pcID(v), v,
				)
				fmt.Fprintf(
					w,
					"<div class=\"wz wz-ret-p%d-s%d-pc-next-v%d\"><label for=\"%s\">></label></div>",
					pc, sp, v, phaseID(phaseRETStart+1),
				)
			}
			fmt.Fprintf(w, "<div class=\"wz wz-ret-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp-1), sp-1)
			fmt.Fprintf(w, "<div class=\"wz wz-ret-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseRETEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-ret-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// SUB labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SUB" {
			continue
		}
		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			fmt.Fprintf(w, "<div class=\"wz wz-sub-p%d-s%d-b0-set\"><label for=\"%s\">b0=0</label></div>", pc, sp, carBitID(0, 0))
			fmt.Fprintf(w, "<div class=\"wz wz-sub-p%d-s%d-b0-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseSUBStart+1))

			for bit := 0; bit < runtimeWordBits; bit++ {
				diffPhase := phaseSUBStart + 1 + bit*2
				borrowPhase := diffPhase + 1

				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						for bin := 0; bin <= 1; bin++ {
							d := diffBit(av, bv, bin)
							bout := borrowBit(av, bv, bin)
							fmt.Fprintf(
								w,
								"<div class=\"wz wz-sub-p%d-s%d-d%d-set%d-a%d-b%d-i%d\"><label for=\"%s\">d%d=%d</label></div>",
								pc, sp, bit, d, av, bv, bin, sumBitID(bit, d), bit, d,
							)
							fmt.Fprintf(
								w,
								"<div class=\"wz wz-sub-p%d-s%d-d%d-next-a%d-b%d-i%d\"><label for=\"%s\">></label></div>",
								pc, sp, bit, av, bv, bin, phaseID(borrowPhase),
							)
							fmt.Fprintf(
								w,
								"<div class=\"wz wz-sub-p%d-s%d-b%d-set%d-a%d-b%d-i%d\"><label for=\"%s\">b%d=%d</label></div>",
								pc, sp, bit+1, bout, av, bv, bin, carBitID(bit+1, bout), bit+1, bout,
							)
							fmt.Fprintf(
								w,
								"<div class=\"wz wz-sub-p%d-s%d-b%d-next-a%d-b%d-i%d\"><label for=\"%s\">></label></div>",
								pc, sp, bit+1, av, bv, bin, phaseID(borrowPhase+1),
							)
						}
					}
				}
			}

			wbStart := phaseSUBStart + 17
			dest := sp - 2
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-sub-p%d-s%d-wb%d-set%d-d%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dest, bit, sv), dest, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-sub-p%d-s%d-wb%d-next-d%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(wbStart+bit+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-sub-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp-1), sp-1)
			fmt.Fprintf(w, "<div class=\"wz wz-sub-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseSUBStart+26))
			fmt.Fprintf(w, "<div class=\"wz wz-sub-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-sub-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseSUBEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-sub-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// AND labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "AND" {
			continue
		}
		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseANDStart + bit
				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						r := av & bv
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-and-p%d-s%d-u%d-set%d-a%d-b%d\"><label for=\"%s\">u%d=%d</label></div>",
							pc, sp, bit, r, av, bv, sumBitID(bit, r), bit, r,
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-and-p%d-s%d-u%d-next-a%d-b%d\"><label for=\"%s\">></label></div>",
							pc, sp, bit, av, bv, phaseID(sumPhase+1),
						)
					}
				}
			}

			wbStart := phaseANDStart + 8
			dest := sp - 2
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-and-p%d-s%d-wb%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dest, bit, sv), dest, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-and-p%d-s%d-wb%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(wbStart+bit+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-and-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp-1), sp-1)
			fmt.Fprintf(w, "<div class=\"wz wz-and-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseANDStart+17))
			fmt.Fprintf(w, "<div class=\"wz wz-and-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-and-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseANDEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-and-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// XOR labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "XOR" {
			continue
		}
		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseXORStart + bit
				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						r := av ^ bv
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-xor-p%d-s%d-u%d-set%d-a%d-b%d\"><label for=\"%s\">u%d=%d</label></div>",
							pc, sp, bit, r, av, bv, sumBitID(bit, r), bit, r,
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-xor-p%d-s%d-u%d-next-a%d-b%d\"><label for=\"%s\">></label></div>",
							pc, sp, bit, av, bv, phaseID(sumPhase+1),
						)
					}
				}
			}

			wbStart := phaseXORStart + 8
			dest := sp - 2
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-xor-p%d-s%d-wb%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dest, bit, sv), dest, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-xor-p%d-s%d-wb%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(wbStart+bit+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-xor-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp-1), sp-1)
			fmt.Fprintf(w, "<div class=\"wz wz-xor-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseXORStart+17))
			fmt.Fprintf(w, "<div class=\"wz wz-xor-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-xor-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseXOREnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-xor-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// OR labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "OR" {
			continue
		}
		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseORStart + bit
				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						r := av | bv
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-or-p%d-s%d-u%d-set%d-a%d-b%d\"><label for=\"%s\">u%d=%d</label></div>",
							pc, sp, bit, r, av, bv, sumBitID(bit, r), bit, r,
						)
						fmt.Fprintf(
							w,
							"<div class=\"wz wz-or-p%d-s%d-u%d-next-a%d-b%d\"><label for=\"%s\">></label></div>",
							pc, sp, bit, av, bv, phaseID(sumPhase+1),
						)
					}
				}
			}

			wbStart := phaseORStart + 8
			dest := sp - 2
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-or-p%d-s%d-wb%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dest, bit, sv), dest, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-or-p%d-s%d-wb%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(wbStart+bit+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-or-p%d-s%d-sp-set\"><label for=\"%s\">sp=%d</label></div>", pc, sp, spID(sp-1), sp-1)
			fmt.Fprintf(w, "<div class=\"wz wz-or-p%d-s%d-sp-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseORStart+17))
			fmt.Fprintf(w, "<div class=\"wz wz-or-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-or-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseOREnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-or-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// NOT labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "NOT" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseNOTStart + bit
				for av := 0; av <= 1; av++ {
					r := 1 - av
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-not-p%d-s%d-u%d-set%d-a%d\"><label for=\"%s\">u%d=%d</label></div>",
						pc, sp, bit, r, av, sumBitID(bit, r), bit, r,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-not-p%d-s%d-u%d-next-a%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, av, phaseID(sumPhase+1),
					)
				}
			}

			wbStart := phaseNOTStart + 8
			dest := sp - 1
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-not-p%d-s%d-wb%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dest, bit, sv), dest, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-not-p%d-s%d-wb%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(wbStart+bit+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-not-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-not-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseNOTEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-not-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// SHL labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SHL" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-shl-p%d-s%d-u0-set0\"><label for=\"%s\">u0=0</label></div>",
				pc, sp, sumBitID(0, 0),
			)
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-shl-p%d-s%d-u0-next\"><label for=\"%s\">></label></div>",
				pc, sp, phaseID(phaseSHLStart+1),
			)

			for bit := 1; bit < runtimeWordBits; bit++ {
				sumPhase := phaseSHLStart + bit
				for av := 0; av <= 1; av++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-shl-p%d-s%d-u%d-set%d-a%d\"><label for=\"%s\">u%d=%d</label></div>",
						pc, sp, bit, av, av, sumBitID(bit, av), bit, av,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-shl-p%d-s%d-u%d-next-a%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, av, phaseID(sumPhase+1),
					)
				}
			}

			wbStart := phaseSHLStart + 8
			dest := sp - 1
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-shl-p%d-s%d-wb%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dest, bit, sv), dest, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-shl-p%d-s%d-wb%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(wbStart+bit+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-shl-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-shl-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseSHLEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-shl-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// SHR labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SHR" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits-1; bit++ {
				sumPhase := phaseSHRStart + bit
				for av := 0; av <= 1; av++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-shr-p%d-s%d-u%d-set%d-a%d\"><label for=\"%s\">u%d=%d</label></div>",
						pc, sp, bit, av, av, sumBitID(bit, av), bit, av,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-shr-p%d-s%d-u%d-next-a%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, av, phaseID(sumPhase+1),
					)
				}
			}

			fmt.Fprintf(
				w,
				"<div class=\"wz wz-shr-p%d-s%d-u7-set0\"><label for=\"%s\">u7=0</label></div>",
				pc, sp, sumBitID(7, 0),
			)
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-shr-p%d-s%d-u7-next\"><label for=\"%s\">></label></div>",
				pc, sp, phaseID(phaseSHRStart+8),
			)

			wbStart := phaseSHRStart + 8
			dest := sp - 1
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-shr-p%d-s%d-wb%d-set%d-u%d\"><label for=\"%s\">w%d.%d=%d</label></div>",
						pc, sp, bit, sv, sv, stkBitID(dest, bit, sv), dest, bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-shr-p%d-s%d-wb%d-next-u%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(wbStart+bit+1),
					)
				}
			}

			fmt.Fprintf(w, "<div class=\"wz wz-shr-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-shr-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseSHREnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-shr-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// JMP labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "JMP" {
			continue
		}
		targetPC := int(words[pc].Operand)
		for sp := 0; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-jmp-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>",
				pc, sp, pcID(targetPC), targetPC,
			)
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-jmp-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>",
				pc, sp, phaseID(phaseJMPEnd),
			)
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-jmp-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// JNZ labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "JNZ" {
			continue
		}
		targetPC := int(words[pc].Operand)
		fallthroughPC := pc + 1
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-jnz-p%d-s%d-pc-set-z\"><label for=\"%s\">pc=%d</label></div>",
				pc, sp, pcID(fallthroughPC), fallthroughPC,
			)
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-jnz-p%d-s%d-pc-next-z\"><label for=\"%s\">></label></div>",
				pc, sp, phaseID(phaseJNZEnd),
			)
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-jnz-p%d-s%d-pc-set-nz\"><label for=\"%s\">pc=%d</label></div>",
				pc, sp, pcID(targetPC), targetPC,
			)
			fmt.Fprintf(
				w,
				"<div class=\"wz wz-jnz-p%d-s%d-pc-next-nz\"><label for=\"%s\">></label></div>",
				pc, sp, phaseID(phaseJNZEnd),
			)
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-jnz-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))

	// SAY labels
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SAY" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				for sv := 0; sv <= 1; sv++ {
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-say-p%d-s%d-b%d-set%d\"><label for=\"%s\">o%d=%d</label></div>",
						pc, sp, bit, sv, outBitID(bit, sv), bit, sv,
					)
					fmt.Fprintf(
						w,
						"<div class=\"wz wz-say-p%d-s%d-b%d-next%d\"><label for=\"%s\">></label></div>",
						pc, sp, bit, sv, phaseID(phaseSAYStart+bit+1),
					)
				}
			}
			fmt.Fprintf(w, "<div class=\"wz wz-say-p%d-s%d-valid-set\"><label for=\"%s\">ov=1</label></div>", pc, sp, outValidID(1))
			fmt.Fprintf(w, "<div class=\"wz wz-say-p%d-s%d-valid-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseSAYStart+9))
			fmt.Fprintf(w, "<div class=\"wz wz-say-p%d-s%d-pc-set\"><label for=\"%s\">pc=%d</label></div>", pc, sp, pcID(pc+1), pc+1)
			fmt.Fprintf(w, "<div class=\"wz wz-say-p%d-s%d-pc-next\"><label for=\"%s\">></label></div>", pc, sp, phaseID(phaseSAYEnd))
		}
	}
	fmt.Fprintf(w, "<div class=\"wz wz-say-reset\"><label for=\"%s\">h0</label></div>", phaseID(phaseDispatch))
}
