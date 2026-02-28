// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import (
	"fmt"
	"strings"
)

func genANDCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "AND" {
			continue
		}
		nextPC := pc + 1

		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			aCell := sp - 2
			bCell := sp - 1

			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseANDStart + bit
				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						r := av & bv
						base := "body" +
							has(opcID(pc)) +
							has(pcID(pc)) +
							has(ospID(sp)) +
							has(spID(sp)) +
							has(phaseID(sumPhase)) +
							has(stkBitID(aCell, bit, av)) +
							has(stkBitID(bCell, bit, bv))

						setSel := base + notHas(sumBitID(bit, r))
						nextSel := base + has(sumBitID(bit, r))
						fmt.Fprintf(
							b,
							"%s .wz-and-p%d-s%d-u%d-set%d-a%d-b%d{visibility:visible;pointer-events:auto}\n",
							setSel, pc, sp, bit, r, av, bv,
						)
						fmt.Fprintf(
							b,
							"%s .wz-and-p%d-s%d-u%d-next-a%d-b%d{visibility:visible;pointer-events:auto}\n",
							nextSel, pc, sp, bit, av, bv,
						)
					}
				}
			}

			wbStart := phaseANDStart + 8
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := wbStart + bit
				for sv := 0; sv <= 1; sv++ {
					wbBase := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(sumBitID(bit, sv))
					wbSet := wbBase + notHas(stkBitID(aCell, bit, sv))
					wbNext := wbBase + has(stkBitID(aCell, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-and-p%d-s%d-wb%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-and-p%d-s%d-wb%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseANDStart+16)) +
				notHas(spID(sp-1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseANDStart+16))
			fmt.Fprintf(b, "%s .wz-and-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-and-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseANDStart+17)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseANDStart+17))
			fmt.Fprintf(b, "%s .wz-and-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-and-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-and-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseANDEnd)),
	)
}

func genXORCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "XOR" {
			continue
		}
		nextPC := pc + 1

		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			aCell := sp - 2
			bCell := sp - 1

			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseXORStart + bit
				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						r := av ^ bv
						base := "body" +
							has(opcID(pc)) +
							has(pcID(pc)) +
							has(ospID(sp)) +
							has(spID(sp)) +
							has(phaseID(sumPhase)) +
							has(stkBitID(aCell, bit, av)) +
							has(stkBitID(bCell, bit, bv))

						setSel := base + notHas(sumBitID(bit, r))
						nextSel := base + has(sumBitID(bit, r))
						fmt.Fprintf(
							b,
							"%s .wz-xor-p%d-s%d-u%d-set%d-a%d-b%d{visibility:visible;pointer-events:auto}\n",
							setSel, pc, sp, bit, r, av, bv,
						)
						fmt.Fprintf(
							b,
							"%s .wz-xor-p%d-s%d-u%d-next-a%d-b%d{visibility:visible;pointer-events:auto}\n",
							nextSel, pc, sp, bit, av, bv,
						)
					}
				}
			}

			wbStart := phaseXORStart + 8
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := wbStart + bit
				for sv := 0; sv <= 1; sv++ {
					wbBase := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(sumBitID(bit, sv))
					wbSet := wbBase + notHas(stkBitID(aCell, bit, sv))
					wbNext := wbBase + has(stkBitID(aCell, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-xor-p%d-s%d-wb%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-xor-p%d-s%d-wb%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseXORStart+16)) +
				notHas(spID(sp-1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseXORStart+16))
			fmt.Fprintf(b, "%s .wz-xor-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-xor-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseXORStart+17)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseXORStart+17))
			fmt.Fprintf(b, "%s .wz-xor-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-xor-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-xor-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseXOREnd)),
	)
}

func genORCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "OR" {
			continue
		}
		nextPC := pc + 1

		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			aCell := sp - 2
			bCell := sp - 1

			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseORStart + bit
				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						r := av | bv
						base := "body" +
							has(opcID(pc)) +
							has(pcID(pc)) +
							has(ospID(sp)) +
							has(spID(sp)) +
							has(phaseID(sumPhase)) +
							has(stkBitID(aCell, bit, av)) +
							has(stkBitID(bCell, bit, bv))

						setSel := base + notHas(sumBitID(bit, r))
						nextSel := base + has(sumBitID(bit, r))
						fmt.Fprintf(
							b,
							"%s .wz-or-p%d-s%d-u%d-set%d-a%d-b%d{visibility:visible;pointer-events:auto}\n",
							setSel, pc, sp, bit, r, av, bv,
						)
						fmt.Fprintf(
							b,
							"%s .wz-or-p%d-s%d-u%d-next-a%d-b%d{visibility:visible;pointer-events:auto}\n",
							nextSel, pc, sp, bit, av, bv,
						)
					}
				}
			}

			wbStart := phaseORStart + 8
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := wbStart + bit
				for sv := 0; sv <= 1; sv++ {
					wbBase := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(sumBitID(bit, sv))
					wbSet := wbBase + notHas(stkBitID(aCell, bit, sv))
					wbNext := wbBase + has(stkBitID(aCell, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-or-p%d-s%d-wb%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-or-p%d-s%d-wb%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseORStart+16)) +
				notHas(spID(sp-1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseORStart+16))
			fmt.Fprintf(b, "%s .wz-or-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-or-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseORStart+17)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseORStart+17))
			fmt.Fprintf(b, "%s .wz-or-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-or-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-or-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseOREnd)),
	)
}

func genNOTCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "NOT" {
			continue
		}
		nextPC := pc + 1

		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			cell := sp - 1

			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseNOTStart + bit
				for av := 0; av <= 1; av++ {
					r := 1 - av
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(stkBitID(cell, bit, av))

					setSel := base + notHas(sumBitID(bit, r))
					nextSel := base + has(sumBitID(bit, r))
					fmt.Fprintf(
						b,
						"%s .wz-not-p%d-s%d-u%d-set%d-a%d{visibility:visible;pointer-events:auto}\n",
						setSel, pc, sp, bit, r, av,
					)
					fmt.Fprintf(
						b,
						"%s .wz-not-p%d-s%d-u%d-next-a%d{visibility:visible;pointer-events:auto}\n",
						nextSel, pc, sp, bit, av,
					)
				}
			}

			wbStart := phaseNOTStart + 8
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := wbStart + bit
				for sv := 0; sv <= 1; sv++ {
					wbBase := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(sumBitID(bit, sv))
					wbSet := wbBase + notHas(stkBitID(cell, bit, sv))
					wbNext := wbBase + has(stkBitID(cell, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-not-p%d-s%d-wb%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-not-p%d-s%d-wb%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseNOTStart+16)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseNOTStart+16))
			fmt.Fprintf(b, "%s .wz-not-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-not-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-not-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseNOTEnd)),
	)
}

func genSHLCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SHL" {
			continue
		}
		nextPC := pc + 1

		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			cell := sp - 1

			// u0 := 0
			u0SetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSHLStart)) +
				notHas(sumBitID(0, 0))
			u0NextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSHLStart)) +
				has(sumBitID(0, 0))
			fmt.Fprintf(b, "%s .wz-shl-p%d-s%d-u0-set0{visibility:visible;pointer-events:auto}\n", u0SetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-shl-p%d-s%d-u0-next{visibility:visible;pointer-events:auto}\n", u0NextSel, pc, sp)

			// ui := a(i-1) for i in 1..7
			for bit := 1; bit < runtimeWordBits; bit++ {
				ph := phaseSHLStart + bit
				srcBit := bit - 1
				for av := 0; av <= 1; av++ {
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(stkBitID(cell, srcBit, av))
					setSel := base + notHas(sumBitID(bit, av))
					nextSel := base + has(sumBitID(bit, av))
					fmt.Fprintf(
						b,
						"%s .wz-shl-p%d-s%d-u%d-set%d-a%d{visibility:visible;pointer-events:auto}\n",
						setSel, pc, sp, bit, av, av,
					)
					fmt.Fprintf(
						b,
						"%s .wz-shl-p%d-s%d-u%d-next-a%d{visibility:visible;pointer-events:auto}\n",
						nextSel, pc, sp, bit, av,
					)
				}
			}

			wbStart := phaseSHLStart + 8
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := wbStart + bit
				for sv := 0; sv <= 1; sv++ {
					wbBase := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(sumBitID(bit, sv))
					wbSet := wbBase + notHas(stkBitID(cell, bit, sv))
					wbNext := wbBase + has(stkBitID(cell, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-shl-p%d-s%d-wb%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-shl-p%d-s%d-wb%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSHLStart+16)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSHLStart+16))
			fmt.Fprintf(b, "%s .wz-shl-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-shl-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-shl-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseSHLEnd)),
	)
}

func genSHRCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SHR" {
			continue
		}
		nextPC := pc + 1

		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			cell := sp - 1

			// ui := a(i+1) for i in 0..6
			for bit := 0; bit < runtimeWordBits-1; bit++ {
				ph := phaseSHRStart + bit
				srcBit := bit + 1
				for av := 0; av <= 1; av++ {
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(stkBitID(cell, srcBit, av))
					setSel := base + notHas(sumBitID(bit, av))
					nextSel := base + has(sumBitID(bit, av))
					fmt.Fprintf(
						b,
						"%s .wz-shr-p%d-s%d-u%d-set%d-a%d{visibility:visible;pointer-events:auto}\n",
						setSel, pc, sp, bit, av, av,
					)
					fmt.Fprintf(
						b,
						"%s .wz-shr-p%d-s%d-u%d-next-a%d{visibility:visible;pointer-events:auto}\n",
						nextSel, pc, sp, bit, av,
					)
				}
			}

			// u7 := 0
			u7SetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSHRStart+7)) +
				notHas(sumBitID(7, 0))
			u7NextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSHRStart+7)) +
				has(sumBitID(7, 0))
			fmt.Fprintf(b, "%s .wz-shr-p%d-s%d-u7-set0{visibility:visible;pointer-events:auto}\n", u7SetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-shr-p%d-s%d-u7-next{visibility:visible;pointer-events:auto}\n", u7NextSel, pc, sp)

			wbStart := phaseSHRStart + 8
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := wbStart + bit
				for sv := 0; sv <= 1; sv++ {
					wbBase := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(sumBitID(bit, sv))
					wbSet := wbBase + notHas(stkBitID(cell, bit, sv))
					wbNext := wbBase + has(stkBitID(cell, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-shr-p%d-s%d-wb%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-shr-p%d-s%d-wb%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSHRStart+16)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSHRStart+16))
			fmt.Fprintf(b, "%s .wz-shr-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-shr-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-shr-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseSHREnd)),
	)
}
