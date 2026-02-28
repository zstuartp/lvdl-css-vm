// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import (
	"fmt"
	"html"
	"io"
	"regexp"
	"sort"
	"strings"

	"lvdl-vm/internal/lvdl"
)

var (
	betweenTagsWSRe = regexp.MustCompile(`>\s+<`)
	wzClassTokenRe  = regexp.MustCompile(`\bwz-[a-z0-9-]+\b`)
)

func minifyHTML(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = betweenTagsWSRe.ReplaceAllString(s, "><")
	s = strings.ReplaceAll(s, " />", "/>")
	return strings.TrimSpace(s)
}

func shortBase36(n int) string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	var out [16]byte
	i := len(out)
	for n > 0 {
		i--
		out[i] = digits[n%36]
		n /= 36
	}
	return string(out[i:])
}

func replaceToken(s, old, new string) string {
	// Class-token safe replacement: old/new only touch full identifier tokens.
	var out strings.Builder
	out.Grow(len(s))
	last := 0
	searchAt := 0
	for {
		idx := strings.Index(s[searchAt:], old)
		if idx < 0 {
			break
		}
		start := searchAt + idx
		end := start + len(old)
		prevOK := start == 0 || !isIdentChar(s[start-1])
		nextOK := end == len(s) || !isIdentChar(s[end])
		if prevOK && nextOK {
			out.WriteString(s[last:start])
			out.WriteString(new)
			last = end
		}
		searchAt = end
	}
	if last == 0 {
		return s
	}
	out.WriteString(s[last:])
	return out.String()
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_'
}

func compressWriteZoneClasses(s string) string {
	preserve := map[string]bool{
		"wz-dispatch": true,
		"wz-next":     true,
		"wz-set":      true,
		"wz-reset":    true,
	}
	uniq := map[string]struct{}{}
	for _, token := range wzClassTokenRe.FindAllString(s, -1) {
		if preserve[token] {
			continue
		}
		uniq[token] = struct{}{}
	}
	if len(uniq) == 0 {
		return s
	}

	old := make([]string, 0, len(uniq))
	for token := range uniq {
		old = append(old, token)
	}
	sort.Strings(old)

	used := map[string]bool{
		"wz":          true,
		"wz-dispatch": true,
		"wz-next":     true,
		"wz-set":      true,
		"wz-reset":    true,
	}
	nextID := 0
	for _, token := range old {
		var short string
		for {
			short = "wz" + shortBase36(nextID)
			nextID++
			if !used[short] {
				break
			}
		}
		used[short] = true
		s = replaceToken(s, token, short)
	}
	return s
}

func writeClockControlsHTML(w io.Writer) {
	io.WriteString(w, "<div class=\"clock\">")
	io.WriteString(w, "<button type=\"button\" id=\"clk-step\">Step</button>")
	io.WriteString(w, "<button type=\"button\" id=\"clk-run\">Run</button>")
	io.WriteString(w, "<button type=\"button\" id=\"clk-stop\">Stop</button>")
	io.WriteString(w, "<button type=\"button\" class=\"clk-speed\" data-ms=\"500\">SLOW</button>")
	io.WriteString(w, "<button type=\"button\" class=\"clk-speed\" data-ms=\"250\">MEDIUM</button>")
	io.WriteString(w, "<button type=\"button\" class=\"clk-speed\" data-ms=\"100\">FAST</button>")
	io.WriteString(w, "<button type=\"button\" class=\"clk-speed\" data-ms=\"10\">TURBO</button>")
	io.WriteString(w, "<button type=\"button\" class=\"clk-speed\" data-ms=\"0\">MAX</button>")
	io.WriteString(w, "<span id=\"clk-state\" class=\"status\">manual</span>")
	io.WriteString(w, "</div>")
}

func writeClockScript(w io.Writer) {
	io.WriteString(w, "<script>")
	io.WriteString(w, `(function(){`)
	io.WriteString(w, `const bus=document.querySelector('.write-bus');`)
	io.WriteString(w, `if(!bus)return;`)
	io.WriteString(w, `const runBtn=document.getElementById('clk-run');`)
	io.WriteString(w, `const stopBtn=document.getElementById('clk-stop');`)
	io.WriteString(w, `const stepBtn=document.getElementById('clk-step');`)
	io.WriteString(w, `const speedBtns=document.querySelectorAll('.clk-speed');`)
	io.WriteString(w, `const state=document.getElementById('clk-state');`)
	io.WriteString(w, `const form=document.querySelector('form');`)
	io.WriteString(w, `const prog=document.querySelector('.program');`)
	io.WriteString(w, `let running=false;`)
	io.WriteString(w, `let timer=0;`)
	io.WriteString(w, `let delayMs=250;`)
	io.WriteString(w, `function setSpeed(ms){`)
	io.WriteString(w, `delayMs=ms;`)
	io.WriteString(w, `for(let i=0;i<speedBtns.length;i++){`)
	io.WriteString(w, `const b=speedBtns[i];`)
	io.WriteString(w, `if(Number(b.dataset.ms)===ms){b.classList.add('speed-active');}else{b.classList.remove('speed-active');}`)
	io.WriteString(w, `}`)
	io.WriteString(w, `}`)
	io.WriteString(w, `function visibleWriteLabels(){`)
	io.WriteString(w, `const out=[];`)
	io.WriteString(w, `const labels=bus.querySelectorAll('.wz label');`)
	io.WriteString(w, `for(let i=0;i<labels.length;i++){`)
	io.WriteString(w, `const label=labels[i];`)
	io.WriteString(w, `const zone=label.parentElement;`)
	io.WriteString(w, `const cs=window.getComputedStyle(zone);`)
	io.WriteString(w, `if(cs.visibility==='visible'&&cs.pointerEvents!=='none'&&cs.display!=='none'){out.push(label);}`)
	io.WriteString(w, `}`)
	io.WriteString(w, `return out;`)
	io.WriteString(w, `}`)
	io.WriteString(w, `function setState(s){if(state)state.textContent=s;}`)
	io.WriteString(w, `function currentPC(){`)
	io.WriteString(w, `const pcs=document.querySelectorAll('input[name="pc"]');`)
	io.WriteString(w, `for(let i=0;i<pcs.length;i++){if(pcs[i].checked){return Number(pcs[i].id.slice(1));}}`)
	io.WriteString(w, `return -1;`)
	io.WriteString(w, `}`)
	io.WriteString(w, `function scrollProgramToPC(){`)
	io.WriteString(w, `if(!prog)return;`)
	io.WriteString(w, `const pc=currentPC();`)
	io.WriteString(w, `if(pc<0)return;`)
	io.WriteString(w, `const row=prog.querySelector('.pl[data-p="'+pc+'"]');`)
	io.WriteString(w, `if(!row)return;`)
	io.WriteString(w, `const rowTop=row.offsetTop;`)
	io.WriteString(w, `const rowBottom=rowTop+row.offsetHeight;`)
	io.WriteString(w, `const viewTop=prog.scrollTop;`)
	io.WriteString(w, `const viewBottom=viewTop+prog.clientHeight;`)
	io.WriteString(w, `if(rowTop<viewTop||rowBottom>viewBottom){`)
	io.WriteString(w, `prog.scrollTop=Math.max(0,rowTop-Math.floor((prog.clientHeight-row.offsetHeight)/2));`)
	io.WriteString(w, `}`)
	io.WriteString(w, `}`)
	io.WriteString(w, `function stepOnce(){`)
	io.WriteString(w, `const labels=visibleWriteLabels();`)
	io.WriteString(w, `if(labels.length!==1){`)
	io.WriteString(w, `if(labels.length===0){setState('halted');}else{setState('conflict');}`)
	io.WriteString(w, `return false;`)
	io.WriteString(w, `}`)
	io.WriteString(w, `labels[0].click();`)
	io.WriteString(w, `scrollProgramToPC();`)
	io.WriteString(w, `return true;`)
	io.WriteString(w, `}`)
	io.WriteString(w, `function stop(reason){`)
	io.WriteString(w, `running=false;`)
	io.WriteString(w, `if(timer){window.clearTimeout(timer);timer=0;}`)
	io.WriteString(w, `setState(reason||'manual');`)
	io.WriteString(w, `}`)
	io.WriteString(w, `function tick(){`)
	io.WriteString(w, `if(!running)return;`)
	io.WriteString(w, `if(!stepOnce()){stop();return;}`)
	io.WriteString(w, `timer=window.setTimeout(tick,delayMs);`)
	io.WriteString(w, `}`)
	io.WriteString(w, `function run(){`)
	io.WriteString(w, `if(running)return;`)
	io.WriteString(w, `running=true;`)
	io.WriteString(w, `setState('running');`)
	io.WriteString(w, `timer=window.setTimeout(tick,0);`)
	io.WriteString(w, `}`)
	io.WriteString(w, `if(stepBtn)stepBtn.addEventListener('click',function(){if(!running)stepOnce();});`)
	io.WriteString(w, `if(runBtn)runBtn.addEventListener('click',run);`)
	io.WriteString(w, `if(stopBtn)stopBtn.addEventListener('click',function(){stop('stopped');});`)
	io.WriteString(w, `for(let i=0;i<speedBtns.length;i++){`)
	io.WriteString(w, `const b=speedBtns[i];`)
	io.WriteString(w, `b.addEventListener('click',function(){setSpeed(Number(b.dataset.ms));});`)
	io.WriteString(w, `}`)
	io.WriteString(w, `setSpeed(250);`)
	io.WriteString(w, `if(form)form.addEventListener('reset',function(){window.setTimeout(function(){stop('manual');scrollProgramToPC();},0);});`)
	io.WriteString(w, `window.setTimeout(scrollProgramToPC,0);`)
	io.WriteString(w, `})();`)
	io.WriteString(w, "</script>")
}

func writeLiveHTML(
	w io.Writer,
	title string,
	spec *lvdl.Spec,
	program []uint16,
	withClock bool,
	opts Options,
) error {
	if title == "" {
		title = "LVDL-VM"
	}

	opts = normalizedOptions(opts)

	if err := validateRuntimeSpec(spec, opts.Profile); err != nil {
		return err
	}

	fixed, err := normalizeProgram(program, opts.Profile)
	if err != nil {
		return err
	}

	decoded, err := decodeProgramWords(spec, fixed, opts.Profile)
	if err != nil {
		return err
	}
	plan := buildRuntimePlan(decoded, opts)

	var doc strings.Builder
	doc.Grow(2 << 20)

	io.WriteString(&doc, "<!doctype html>\n<html lang=\"en\">\n<head>\n")
	io.WriteString(&doc, "<meta charset=\"utf-8\" />\n")
	io.WriteString(&doc, "<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\" />\n")
	fmt.Fprintf(&doc, "<title>%s</title>\n", html.EscapeString(title))
	io.WriteString(&doc, "<style>\n")
	io.WriteString(&doc, genRuntimeCSS(decoded, plan))
	io.WriteString(&doc, "\n</style>\n</head>\n<body>\n")

	themeRadios(&doc)
	io.WriteString(&doc, "<form>\n")

	writeStateRadiosHTML(&doc, plan)

	io.WriteString(&doc, "<div class=\"app\">\n")
	io.WriteString(&doc, "<div class=\"title\">\n")
	fmt.Fprintf(&doc, "<h1>%s</h1>\n", html.EscapeString(title))
	io.WriteString(&doc, "<span class=\"badge-css\">CSS runtime</span>\n")
	if withClock {
		io.WriteString(&doc, "<span class=\"badge-js\">JS clock</span>\n")
	}
	io.WriteString(&doc, "<span class=\"status status-running\">RUNNING</span>\n")
	io.WriteString(&doc, "<span class=\"status status-halted\">HALTED</span>\n")
	io.WriteString(&doc, "<span>i:</span><span class=\"id\">")
	for pc := 0; pc < plan.visibleProgramWords(); pc++ {
		fmt.Fprintf(&doc, "<span class=\"i%d\" style=\"display:none\">%s</span>", pc, html.EscapeString(decoded[pc].Text))
	}
	io.WriteString(&doc, "</span>\n")
	if withClock {
		writeClockControlsHTML(&doc)
	}
	io.WriteString(&doc, "<input type=\"reset\" value=\"Reset\" class=\"status\" style=\"cursor:pointer\">\n")
	themeToggleHTML(&doc)
	io.WriteString(&doc, "</div>\n")

	io.WriteString(&doc, "<div class=\"panel\">\n")
	io.WriteString(&doc, "<div class=\"kv\">\n")
	io.WriteString(&doc, "<div class=\"box\"><div class=\"label\">PC</div><div class=\"value pd\">")
	for pc := 0; pc < plan.visibleProgramWords(); pc++ {
		fmt.Fprintf(&doc, "<span class=\"p%d\">%d</span>", pc, pc)
	}
	io.WriteString(&doc, "</div></div>\n")
	io.WriteString(&doc, "<div class=\"box\"><div class=\"label\">SP</div><div class=\"value sd\">")
	for sp := 0; sp <= plan.profile.StackWords; sp++ {
		fmt.Fprintf(&doc, "<span class=\"s%d\">%d</span>", sp, sp)
	}
	io.WriteString(&doc, "</div></div>\n")
	io.WriteString(&doc, "</div>\n")

	io.WriteString(&doc, "<h3>Write Bus</h3>\n")
	io.WriteString(&doc, "<div class=\"write-bus\">\n")
	genWriteZonesHTML(&doc, decoded, plan)
	io.WriteString(&doc, "</div>\n")

	io.WriteString(&doc, "<h3>Stack Bits</h3>\n<div class=\"stack\">\n")
	visibleCells := plan.visibleStackCells()
	for cell := 0; cell < visibleCells; cell++ {
		fmt.Fprintf(&doc, "<div class=\"stk-row\">[%d] ", cell)
		for bit := runtimeWordBits - 1; bit >= 0; bit-- {
			fmt.Fprintf(&doc, "<span class=\"bit\"><span class=\"k%d%d0\">0</span><span class=\"k%d%d1\">1</span></span>", cell, bit, cell, bit)
		}
		fmt.Fprintf(&doc, "<span class=\"stk-dec d%d\"></span>", cell)
		io.WriteString(&doc, "</div>\n")
	}
	io.WriteString(&doc, "</div>\n")

	io.WriteString(&doc, "<h3>Output</h3><div class=\"ov\"></div>\n")
	io.WriteString(&doc, "</div>\n")

	io.WriteString(&doc, "<div class=\"panel\">\n<h3>Program</h3><div class=\"program\">\n")
	for pc := 0; pc < plan.visibleProgramWords(); pc++ {
		fmt.Fprintf(
			&doc,
			"<div class=\"pl\" data-p=\"%d\">%d:%04X %s</div>\n",
			pc, pc, fixed[pc], html.EscapeString(decoded[pc].Text),
		)
	}
	io.WriteString(&doc, "</div>\n</div>\n")

	io.WriteString(&doc, "</div>\n") // app
	io.WriteString(&doc, "</form>\n")
	if withClock {
		writeClockScript(&doc)
	}
	io.WriteString(&doc, "</body>\n</html>\n")

	out := doc.String()
	out = compressWriteZoneClasses(out)
	out = minifyHTML(out)
	_, err = io.WriteString(w, out)
	return err
}

// WriteLivePureHTML emits the runtime CSS machine with manual stepping only.
func WriteLivePureHTML(w io.Writer, title string, spec *lvdl.Spec, program []uint16) error {
	return WriteLivePureHTMLWithOptions(w, title, spec, program, Options{})
}

// WriteLiveClockHTML emits the runtime CSS machine with a click-only JS clock.
func WriteLiveClockHTML(w io.Writer, title string, spec *lvdl.Spec, program []uint16) error {
	return WriteLiveClockHTMLWithOptions(w, title, spec, program, Options{})
}

// WriteLivePureHTMLWithOptions emits the runtime CSS machine with manual stepping only.
func WriteLivePureHTMLWithOptions(
	w io.Writer,
	title string,
	spec *lvdl.Spec,
	program []uint16,
	opts Options,
) error {
	return writeLiveHTML(w, title, spec, program, false, opts)
}

// WriteLiveClockHTMLWithOptions emits the runtime CSS machine with a click-only JS clock.
func WriteLiveClockHTMLWithOptions(
	w io.Writer,
	title string,
	spec *lvdl.Spec,
	program []uint16,
	opts Options,
) error {
	return writeLiveHTML(w, title, spec, program, true, opts)
}
