// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import (
	"fmt"
	"strings"
)

func genPSHCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "PSH" {
			continue
		}
		imm := int(words[pc].Operand)
		nextPC := pc + 1

		for sp := 0; sp < runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phasePSHStart + bit
				bitVal := (imm >> bit) & 1
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
				fmt.Fprintf(b, "%s .wz-psh-p%d-s%d-b%d-set%d{visibility:visible;pointer-events:auto}\n", setSel, pc, sp, bit, bitVal)
				fmt.Fprintf(b, "%s .wz-psh-p%d-s%d-b%d-next{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp, bit)
			}

			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phasePSHStart+8)) +
				notHas(spID(sp+1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phasePSHStart+8))
			fmt.Fprintf(b, "%s .wz-psh-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-psh-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phasePSHStart+9)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phasePSHStart+9))
			fmt.Fprintf(b, "%s .wz-psh-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-psh-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-psh-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phasePSHEnd)),
	)
}

func genDUPCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "DUP" {
			continue
		}
		nextPC := pc + 1

		for sp := 1; sp < runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			src := sp - 1
			dst := sp

			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseDUPStart + bit
				for sv := 0; sv <= 1; sv++ {
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(stkBitID(src, bit, sv))

					setSel := base + notHas(stkBitID(dst, bit, sv))
					nextSel := base + has(stkBitID(dst, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-dup-p%d-s%d-b%d-set%d-s%d{visibility:visible;pointer-events:auto}\n",
						setSel, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-dup-p%d-s%d-b%d-next-s%d{visibility:visible;pointer-events:auto}\n",
						nextSel, pc, sp, bit, sv,
					)
				}
			}

			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseDUPStart+8)) +
				notHas(spID(sp+1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseDUPStart+8))
			fmt.Fprintf(b, "%s .wz-dup-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-dup-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseDUPStart+9)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseDUPStart+9))
			fmt.Fprintf(b, "%s .wz-dup-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-dup-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-dup-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseDUPEnd)),
	)
}

func genPOPCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "POP" {
			continue
		}
		nextPC := pc + 1

		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phasePOPStart)) +
				notHas(spID(sp-1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phasePOPStart))
			fmt.Fprintf(b, "%s .wz-pop-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-pop-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phasePOPStart+1)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phasePOPStart+1))
			fmt.Fprintf(b, "%s .wz-pop-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-pop-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-pop-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phasePOPEnd)),
	)
}

func genSWPCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SWP" {
			continue
		}
		nextPC := pc + 1

		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			top := sp - 1
			under := sp - 2

			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseSWPStart + bit
				for sv := 0; sv <= 1; sv++ {
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(stkBitID(top, bit, sv))
					setSel := base + notHas(sumBitID(bit, sv))
					nextSel := base + has(sumBitID(bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-swp-p%d-s%d-u%d-set%d-s%d{visibility:visible;pointer-events:auto}\n",
						setSel, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-swp-p%d-s%d-u%d-next-s%d{visibility:visible;pointer-events:auto}\n",
						nextSel, pc, sp, bit, sv,
					)
				}
			}

			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseSWPStart + 8 + bit
				for sv := 0; sv <= 1; sv++ {
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(stkBitID(under, bit, sv))
					setSel := base + notHas(stkBitID(top, bit, sv))
					nextSel := base + has(stkBitID(top, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-swp-p%d-s%d-top%d-set%d-s%d{visibility:visible;pointer-events:auto}\n",
						setSel, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-swp-p%d-s%d-top%d-next-s%d{visibility:visible;pointer-events:auto}\n",
						nextSel, pc, sp, bit, sv,
					)
				}
			}

			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseSWPStart + 16 + bit
				for sv := 0; sv <= 1; sv++ {
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(sumBitID(bit, sv))
					setSel := base + notHas(stkBitID(under, bit, sv))
					nextSel := base + has(stkBitID(under, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-swp-p%d-s%d-low%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						setSel, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-swp-p%d-s%d-low%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						nextSel, pc, sp, bit, sv,
					)
				}
			}

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSWPStart+24)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSWPStart+24))
			fmt.Fprintf(b, "%s .wz-swp-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-swp-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-swp-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseSWPEnd)),
	)
}

func genOVRCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "OVR" {
			continue
		}
		nextPC := pc + 1

		for sp := 2; sp < runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			src := sp - 2
			dst := sp
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseOVRStart + bit
				for sv := 0; sv <= 1; sv++ {
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(stkBitID(src, bit, sv))
					setSel := base + notHas(stkBitID(dst, bit, sv))
					nextSel := base + has(stkBitID(dst, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-ovr-p%d-s%d-b%d-set%d-s%d{visibility:visible;pointer-events:auto}\n",
						setSel, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-ovr-p%d-s%d-b%d-next-s%d{visibility:visible;pointer-events:auto}\n",
						nextSel, pc, sp, bit, sv,
					)
				}
			}

			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseOVRStart+8)) +
				notHas(spID(sp+1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseOVRStart+8))
			fmt.Fprintf(b, "%s .wz-ovr-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-ovr-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseOVRStart+9)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp+1)) +
				has(phaseID(phaseOVRStart+9))
			fmt.Fprintf(b, "%s .wz-ovr-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-ovr-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-ovr-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseOVREnd)),
	)
}
