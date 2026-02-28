// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"lvdl-vm/internal/asm"
	"lvdl-vm/internal/emit"
	"lvdl-vm/internal/lvdl"
	"lvdl-vm/internal/profile"
)

func main() {
	var specPath, asmPath, outPath, title, mode, opt, profileName string

	flag.StringVar(&specPath, "spec", "", "path to .lvdl spec")
	flag.StringVar(&asmPath, "asm", "", "path to .asm program")
	flag.StringVar(&outPath, "out", "out.html", "output HTML path")
	flag.StringVar(&title, "title", "", "HTML title")
	flag.StringVar(&mode, "mode", "live-pure-css", "output mode: live-pure-css | live-js-clock")
	flag.StringVar(&opt, "opt", string(emit.OptimizationControlState), "optimization mode: control-state | none")
	flag.StringVar(&profileName, "profile", profile.Runtime8.Name, "runtime profile: runtime8 | runtime8-tiny")
	flag.Parse()

	if specPath == "" || asmPath == "" {
		fmt.Fprintln(
			os.Stderr,
			"usage: lvdlc -spec ./examples/standard.lvdl -asm ./examples/add_demo.asm [-mode live-pure-css|live-js-clock] [-profile runtime8|runtime8-tiny] [-opt control-state|none] [-out ./out.html]",
		)
		os.Exit(2)
	}

	if mode != "live-pure-css" && mode != "live-js-clock" {
		fmt.Fprintf(os.Stderr, "unknown mode %q (supported: live-pure-css, live-js-clock)\n", mode)
		os.Exit(2)
	}
	if opt != string(emit.OptimizationControlState) && opt != string(emit.OptimizationNone) {
		fmt.Fprintf(
			os.Stderr,
			"unknown opt %q (supported: %s, %s)\n",
			opt,
			emit.OptimizationControlState,
			emit.OptimizationNone,
		)
		os.Exit(2)
	}
	prof, err := profile.Resolve(profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	spec, err := lvdl.LoadFile(specPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spec error:", err)
		os.Exit(1)
	}
	if err := validateCompileSpec(spec, prof); err != nil {
		fmt.Fprintln(os.Stderr, "spec error:", err)
		os.Exit(1)
	}

	f, err := os.Open(asmPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "asm open error:", err)
		os.Exit(1)
	}
	defer f.Close()

	prog, err := asm.AssembleWithOptions(
		f,
		spec.ISA,
		asm.Options{MaxProgramPC: prof.MaxProgramPC()},
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "asm error:", err)
		os.Exit(1)
	}

	out, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "out create error:", err)
		os.Exit(1)
	}
	defer out.Close()

	if title == "" {
		title = "LVDL-VM"
	}
	opts := emit.Options{
		Optimization: emit.OptimizationMode(opt),
		Profile:      prof,
	}

	switch mode {
	case "live-pure-css":
		if err := emit.WriteLivePureHTMLWithOptions(out, title, spec, prog.Words, opts); err != nil {
			fmt.Fprintln(os.Stderr, "emit error:", err)
			os.Exit(1)
		}
	case "live-js-clock":
		if err := emit.WriteLiveClockHTMLWithOptions(out, title, spec, prog.Words, opts); err != nil {
			fmt.Fprintln(os.Stderr, "emit error:", err)
			os.Exit(1)
		}
	}

	fmt.Println("wrote", outPath)
}

func validateCompileSpec(spec *lvdl.Spec, prof profile.Machine) error {
	if spec.Machine.WordBits != prof.WordBits {
		return fmt.Errorf("expected (%s) (word %d), got %d", prof.Name, prof.WordBits, spec.Machine.WordBits)
	}
	if spec.ISA.Encoding.OpcodeBits != prof.OpcodeBits || spec.ISA.Encoding.OperandBits != prof.OperandBits {
		return fmt.Errorf(
			"expected (%s) encoding %d/%d, got %d/%d",
			prof.Name,
			prof.OpcodeBits,
			prof.OperandBits,
			spec.ISA.Encoding.OpcodeBits,
			spec.ISA.Encoding.OperandBits,
		)
	}
	if len(spec.Machine.Stacks) == 0 {
		return fmt.Errorf("expected one stack declaration")
	}
	stk := spec.Machine.Stacks[0]
	if stk.Words != prof.StackWords || stk.Bits != prof.StackBits {
		return fmt.Errorf(
			"expected (%s) first stack to be %d words of %d bits, got %d words / %d bits",
			prof.Name,
			prof.StackWords,
			prof.StackBits,
			stk.Words,
			stk.Bits,
		)
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
		return fmt.Errorf("expected (%s) reg PC %d, got %d", prof.Name, prof.PCBits, pcBits)
	}
	if spBits != prof.SPBits {
		return fmt.Errorf("expected (%s) reg SP %d, got %d", prof.Name, prof.SPBits, spBits)
	}
	return nil
}
