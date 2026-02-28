// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import (
	"fmt"
	"strings"
)

func genCLLCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
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
				target := stkBitID(sp, bit, bitVal)
				setSel := "body" +
					has(opcID(pc)) +
					has(pcID(pc)) +
					has(ospID(sp)) +
					has(spID(sp)) +
					has(phaseID(ph)) +
					notHas(target)
				nextSel := "body" +
					has(opcID(pc)) +
					has(pcID(pc)) +
					has(ospID(sp)) +
					has(spID(sp)) +
					has(phaseID(ph)) +
					has(target)
				fmt.Fprintf(b, "%s .wz-cll-p%d-s%d-b%d-set%d{visibility:visible;pointer-events:auto}\n", setSel, pc, sp, bit, bitVal)
				fmt.Fprintf(b, "%s .wz-cll-p%d-s%d-b%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp, bit)
			}

			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseCLLStart+8)) +
				notHas(spID(sp+1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseCLLStart+8))
			fmt.Fprintf(b, "%s .wz-cll-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-cll-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseCLLStart+9)) +
				notHas(pcID(targetPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(targetPC)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseCLLStart+9))
			fmt.Fprintf(b, "%s .wz-cll-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-cll-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-cll-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseCLLEnd)),
	)
}

func genRETCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "RET" {
			continue
		}
		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			top := sp - 1
			for targetPC := 0; targetPC < plan.profile.ProgramWords; targetPC++ {
				base := "body" +
					has(opcID(pc)) +
					has(pcID(pc)) +
					has(ospID(sp)) +
					has(spID(sp)) +
					has(phaseID(phaseRETStart)) +
					stackValueSel(top, targetPC)

				setSel := base + notHas(pcID(targetPC))
				nextSel := "body" +
					has(opcID(pc)) +
					has(pcID(targetPC)) +
					has(ospID(sp)) +
					has(spID(sp)) +
					has(phaseID(phaseRETStart)) +
					stackValueSel(top, targetPC)

				fmt.Fprintf(
					b,
					"%s .wz-ret-p%d-s%d-pc-set-v%d{visibility:visible;pointer-events:auto}\n",
					setSel, pc, sp, targetPC,
				)
				fmt.Fprintf(
					b,
					"%s .wz-ret-p%d-s%d-pc-next-v%d{visibility:visible;pointer-events:auto}\n",
					nextSel, pc, sp, targetPC,
				)
			}

			spSetSel := "body" +
				has(opcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseRETStart+1)) +
				notHas(spID(sp-1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseRETStart+1))
			fmt.Fprintf(b, "%s .wz-ret-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-ret-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-ret-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseRETEnd)),
	)
}

func genJMPCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "JMP" {
			continue
		}
		targetPC := int(words[pc].Operand)

		for sp := 0; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseJMPStart)) +
				notHas(pcID(targetPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(targetPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseJMPStart))
			fmt.Fprintf(b, "%s .wz-jmp-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-jmp-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-jmp-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseJMPEnd)),
	)
}

func genJNZCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
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
			top := sp - 1
			zeroSetBase := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseJNZStart))
			zeroNextBase := "body" +
				has(opcID(pc)) +
				has(pcID(fallthroughPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseJNZStart))
			for bit := 0; bit < runtimeWordBits; bit++ {
				zeroSetBase += has(stkBitID(top, bit, 0))
				zeroNextBase += has(stkBitID(top, bit, 0))
			}

			zeroSetSel := zeroSetBase + notHas(pcID(fallthroughPC))
			zeroNextSel := zeroNextBase
			fmt.Fprintf(b, "%s .wz-jnz-p%d-s%d-pc-set-z{visibility:visible;pointer-events:auto}\n", zeroSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-jnz-p%d-s%d-pc-next-z{visibility:visible;pointer-events:auto}\n", zeroNextSel, pc, sp)

			for bit := 0; bit < runtimeWordBits; bit++ {
				nzSetBase := "body" +
					has(opcID(pc)) +
					has(pcID(pc)) +
					has(ospID(sp)) +
					has(spID(sp)) +
					has(phaseID(phaseJNZStart)) +
					has(stkBitID(top, bit, 1))
				nzNextBase := "body" +
					has(opcID(pc)) +
					has(pcID(targetPC)) +
					has(ospID(sp)) +
					has(spID(sp)) +
					has(phaseID(phaseJNZStart)) +
					has(stkBitID(top, bit, 1))
				nzSetSel := nzSetBase + notHas(pcID(targetPC))
				nzNextSel := nzNextBase
				fmt.Fprintf(b, "%s .wz-jnz-p%d-s%d-pc-set-nz{visibility:visible;pointer-events:auto}\n", nzSetSel, pc, sp)
				fmt.Fprintf(b, "%s .wz-jnz-p%d-s%d-pc-next-nz{visibility:visible;pointer-events:auto}\n", nzNextSel, pc, sp)
			}
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-jnz-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseJNZEnd)),
	)
}
