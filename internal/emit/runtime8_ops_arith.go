// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import (
	"fmt"
	"strings"
)

func sumBit(a, bv, c int) int {
	return a ^ bv ^ c
}

func carryBit(a, bv, c int) int {
	if (a == 1 && bv == 1) || (a == 1 && c == 1) || (bv == 1 && c == 1) {
		return 1
	}
	return 0
}

func diffBit(a, bv, borrowIn int) int {
	return a ^ bv ^ borrowIn
}

func borrowBit(a, bv, borrowIn int) int {
	if a-bv-borrowIn < 0 {
		return 1
	}
	return 0
}

func genADDCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "ADD" {
			continue
		}
		nextPC := pc + 1

		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			aCell := sp - 2
			bCell := sp - 1

			// car0 := 0
			car0SetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseADDStart)) +
				notHas(carBitID(0, 0))
			car0NextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseADDStart)) +
				has(carBitID(0, 0))
			fmt.Fprintf(b, "%s .wz-add-p%d-s%d-car0-set{visibility:visible;pointer-events:auto}\n", car0SetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-add-p%d-s%d-car0-next{visibility:visible;pointer-events:auto}\n", car0NextSel, pc, sp)

			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseADDStart + 1 + bit*2
				carPhase := sumPhase + 1

				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						for cv := 0; cv <= 1; cv++ {
							s := sumBit(av, bv, cv)
							cnext := carryBit(av, bv, cv)

							sumBase := "body" +
								has(opcID(pc)) +
								has(pcID(pc)) +
								has(ospID(sp)) +
								has(spID(sp)) +
								has(phaseID(sumPhase)) +
								has(stkBitID(aCell, bit, av)) +
								has(stkBitID(bCell, bit, bv)) +
								has(carBitID(bit, cv))

							sumSet := sumBase + notHas(sumBitID(bit, s))
							sumNext := sumBase + has(sumBitID(bit, s))
							fmt.Fprintf(
								b,
								"%s .wz-add-p%d-s%d-sum%d-set%d-a%d-b%d-c%d{visibility:visible;pointer-events:auto}\n",
								sumSet, pc, sp, bit, s, av, bv, cv,
							)
							fmt.Fprintf(
								b,
								"%s .wz-add-p%d-s%d-sum%d-next-a%d-b%d-c%d{visibility:visible;pointer-events:auto}\n",
								sumNext, pc, sp, bit, av, bv, cv,
							)

							carBase := "body" +
								has(opcID(pc)) +
								has(pcID(pc)) +
								has(ospID(sp)) +
								has(spID(sp)) +
								has(phaseID(carPhase)) +
								has(stkBitID(aCell, bit, av)) +
								has(stkBitID(bCell, bit, bv)) +
								has(carBitID(bit, cv)) +
								has(sumBitID(bit, s))

							carSet := carBase + notHas(carBitID(bit+1, cnext))
							carNext := carBase + has(carBitID(bit+1, cnext))
							fmt.Fprintf(
								b,
								"%s .wz-add-p%d-s%d-car%d-set%d-a%d-b%d-c%d{visibility:visible;pointer-events:auto}\n",
								carSet, pc, sp, bit+1, cnext, av, bv, cv,
							)
							fmt.Fprintf(
								b,
								"%s .wz-add-p%d-s%d-car%d-next-a%d-b%d-c%d{visibility:visible;pointer-events:auto}\n",
								carNext, pc, sp, bit+1, av, bv, cv,
							)
						}
					}
				}
			}

			// Writeback sum bits to destination cell (aCell).
			wbStart := phaseADDStart + 17 // 37
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
						"%s .wz-add-p%d-s%d-wb%d-set%d-sum%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-add-p%d-s%d-wb%d-next-sum%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			// SP := SP-1
			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseADDStart+25)) +
				notHas(spID(sp-1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseADDStart+25))
			fmt.Fprintf(b, "%s .wz-add-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-add-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			// PC := PC+1
			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseADDStart+26)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseADDStart+26))
			fmt.Fprintf(b, "%s .wz-add-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-add-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-add-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseADDEnd)),
	)
}

func genINCCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "INC" {
			continue
		}
		nextPC := pc + 1

		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			top := sp - 1

			// car0 := 1
			car0SetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseINCStart)) +
				notHas(carBitID(0, 1))
			car0NextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseINCStart)) +
				has(carBitID(0, 1))
			fmt.Fprintf(b, "%s .wz-inc-p%d-s%d-c0-set{visibility:visible;pointer-events:auto}\n", car0SetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-inc-p%d-s%d-c0-next{visibility:visible;pointer-events:auto}\n", car0NextSel, pc, sp)

			for bit := 0; bit < runtimeWordBits; bit++ {
				sumPhase := phaseINCStart + 1 + bit*2
				carPhase := sumPhase + 1
				for av := 0; av <= 1; av++ {
					for cv := 0; cv <= 1; cv++ {
						s := sumBit(av, 0, cv)
						cnext := carryBit(av, 0, cv)

						sumBase := "body" +
							has(opcID(pc)) +
							has(pcID(pc)) +
							has(ospID(sp)) +
							has(spID(sp)) +
							has(phaseID(sumPhase)) +
							has(stkBitID(top, bit, av)) +
							has(carBitID(bit, cv))
						sumSet := sumBase + notHas(sumBitID(bit, s))
						sumNext := sumBase + has(sumBitID(bit, s))
						fmt.Fprintf(
							b,
							"%s .wz-inc-p%d-s%d-u%d-set%d-a%d-c%d{visibility:visible;pointer-events:auto}\n",
							sumSet, pc, sp, bit, s, av, cv,
						)
						fmt.Fprintf(
							b,
							"%s .wz-inc-p%d-s%d-u%d-next-a%d-c%d{visibility:visible;pointer-events:auto}\n",
							sumNext, pc, sp, bit, av, cv,
						)

						carBase := "body" +
							has(opcID(pc)) +
							has(pcID(pc)) +
							has(ospID(sp)) +
							has(spID(sp)) +
							has(phaseID(carPhase)) +
							has(stkBitID(top, bit, av)) +
							has(carBitID(bit, cv)) +
							has(sumBitID(bit, s))
						carSet := carBase + notHas(carBitID(bit+1, cnext))
						carNext := carBase + has(carBitID(bit+1, cnext))
						fmt.Fprintf(
							b,
							"%s .wz-inc-p%d-s%d-c%d-set%d-a%d-c%d{visibility:visible;pointer-events:auto}\n",
							carSet, pc, sp, bit+1, cnext, av, cv,
						)
						fmt.Fprintf(
							b,
							"%s .wz-inc-p%d-s%d-c%d-next-a%d-c%d{visibility:visible;pointer-events:auto}\n",
							carNext, pc, sp, bit+1, av, cv,
						)
					}
				}
			}

			wbStart := phaseINCStart + 17
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
					wbSet := wbBase + notHas(stkBitID(top, bit, sv))
					wbNext := wbBase + has(stkBitID(top, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-inc-p%d-s%d-wb%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-inc-p%d-s%d-wb%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseINCStart+25)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseINCStart+25))
			fmt.Fprintf(b, "%s .wz-inc-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-inc-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}
	fmt.Fprintf(
		b,
		"body%s .wz-inc-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseINCEnd)),
	)
}

func genDECCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "DEC" {
			continue
		}
		nextPC := pc + 1

		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			top := sp - 1

			// borrow0 := 1
			b0SetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseDECStart)) +
				notHas(carBitID(0, 1))
			b0NextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseDECStart)) +
				has(carBitID(0, 1))
			fmt.Fprintf(b, "%s .wz-dec-p%d-s%d-c0-set{visibility:visible;pointer-events:auto}\n", b0SetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-dec-p%d-s%d-c0-next{visibility:visible;pointer-events:auto}\n", b0NextSel, pc, sp)

			for bit := 0; bit < runtimeWordBits; bit++ {
				diffPhase := phaseDECStart + 1 + bit*2
				borrowPhase := diffPhase + 1
				for av := 0; av <= 1; av++ {
					for bin := 0; bin <= 1; bin++ {
						d := diffBit(av, 0, bin)
						bout := borrowBit(av, 0, bin)

						diffBase := "body" +
							has(opcID(pc)) +
							has(pcID(pc)) +
							has(ospID(sp)) +
							has(spID(sp)) +
							has(phaseID(diffPhase)) +
							has(stkBitID(top, bit, av)) +
							has(carBitID(bit, bin))
						diffSet := diffBase + notHas(sumBitID(bit, d))
						diffNext := diffBase + has(sumBitID(bit, d))
						fmt.Fprintf(
							b,
							"%s .wz-dec-p%d-s%d-u%d-set%d-a%d-c%d{visibility:visible;pointer-events:auto}\n",
							diffSet, pc, sp, bit, d, av, bin,
						)
						fmt.Fprintf(
							b,
							"%s .wz-dec-p%d-s%d-u%d-next-a%d-c%d{visibility:visible;pointer-events:auto}\n",
							diffNext, pc, sp, bit, av, bin,
						)

						borrowBase := "body" +
							has(opcID(pc)) +
							has(pcID(pc)) +
							has(ospID(sp)) +
							has(spID(sp)) +
							has(phaseID(borrowPhase)) +
							has(stkBitID(top, bit, av)) +
							has(carBitID(bit, bin)) +
							has(sumBitID(bit, d))
						borrowSet := borrowBase + notHas(carBitID(bit+1, bout))
						borrowNext := borrowBase + has(carBitID(bit+1, bout))
						fmt.Fprintf(
							b,
							"%s .wz-dec-p%d-s%d-c%d-set%d-a%d-c%d{visibility:visible;pointer-events:auto}\n",
							borrowSet, pc, sp, bit+1, bout, av, bin,
						)
						fmt.Fprintf(
							b,
							"%s .wz-dec-p%d-s%d-c%d-next-a%d-c%d{visibility:visible;pointer-events:auto}\n",
							borrowNext, pc, sp, bit+1, av, bin,
						)
					}
				}
			}

			wbStart := phaseDECStart + 17
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
					wbSet := wbBase + notHas(stkBitID(top, bit, sv))
					wbNext := wbBase + has(stkBitID(top, bit, sv))
					fmt.Fprintf(
						b,
						"%s .wz-dec-p%d-s%d-wb%d-set%d-u%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-dec-p%d-s%d-wb%d-next-u%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseDECStart+25)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseDECStart+25))
			fmt.Fprintf(b, "%s .wz-dec-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-dec-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}
	fmt.Fprintf(
		b,
		"body%s .wz-dec-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseDECEnd)),
	)
}

func genSUBCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SUB" {
			continue
		}
		nextPC := pc + 1

		for sp := 2; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			aCell := sp - 2
			bCell := sp - 1

			// b0 := 0
			b0SetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSUBStart)) +
				notHas(carBitID(0, 0))
			b0NextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSUBStart)) +
				has(carBitID(0, 0))
			fmt.Fprintf(b, "%s .wz-sub-p%d-s%d-b0-set{visibility:visible;pointer-events:auto}\n", b0SetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-sub-p%d-s%d-b0-next{visibility:visible;pointer-events:auto}\n", b0NextSel, pc, sp)

			for bit := 0; bit < runtimeWordBits; bit++ {
				diffPhase := phaseSUBStart + 1 + bit*2
				borrowPhase := diffPhase + 1

				for av := 0; av <= 1; av++ {
					for bv := 0; bv <= 1; bv++ {
						for bin := 0; bin <= 1; bin++ {
							d := diffBit(av, bv, bin)
							bout := borrowBit(av, bv, bin)

							diffBase := "body" +
								has(opcID(pc)) +
								has(pcID(pc)) +
								has(ospID(sp)) +
								has(spID(sp)) +
								has(phaseID(diffPhase)) +
								has(stkBitID(aCell, bit, av)) +
								has(stkBitID(bCell, bit, bv)) +
								has(carBitID(bit, bin))

							diffSet := diffBase + notHas(sumBitID(bit, d))
							diffNext := diffBase + has(sumBitID(bit, d))
							fmt.Fprintf(
								b,
								"%s .wz-sub-p%d-s%d-d%d-set%d-a%d-b%d-i%d{visibility:visible;pointer-events:auto}\n",
								diffSet, pc, sp, bit, d, av, bv, bin,
							)
							fmt.Fprintf(
								b,
								"%s .wz-sub-p%d-s%d-d%d-next-a%d-b%d-i%d{visibility:visible;pointer-events:auto}\n",
								diffNext, pc, sp, bit, av, bv, bin,
							)

							borrowBase := "body" +
								has(opcID(pc)) +
								has(pcID(pc)) +
								has(ospID(sp)) +
								has(spID(sp)) +
								has(phaseID(borrowPhase)) +
								has(stkBitID(aCell, bit, av)) +
								has(stkBitID(bCell, bit, bv)) +
								has(carBitID(bit, bin)) +
								has(sumBitID(bit, d))

							borrowSet := borrowBase + notHas(carBitID(bit+1, bout))
							borrowNext := borrowBase + has(carBitID(bit+1, bout))
							fmt.Fprintf(
								b,
								"%s .wz-sub-p%d-s%d-b%d-set%d-a%d-b%d-i%d{visibility:visible;pointer-events:auto}\n",
								borrowSet, pc, sp, bit+1, bout, av, bv, bin,
							)
							fmt.Fprintf(
								b,
								"%s .wz-sub-p%d-s%d-b%d-next-a%d-b%d-i%d{visibility:visible;pointer-events:auto}\n",
								borrowNext, pc, sp, bit+1, av, bv, bin,
							)
						}
					}
				}
			}

			// Writeback diff bits to destination cell (aCell).
			wbStart := phaseSUBStart + 17
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
						"%s .wz-sub-p%d-s%d-wb%d-set%d-d%d{visibility:visible;pointer-events:auto}\n",
						wbSet, pc, sp, bit, sv, sv,
					)
					fmt.Fprintf(
						b,
						"%s .wz-sub-p%d-s%d-wb%d-next-d%d{visibility:visible;pointer-events:auto}\n",
						wbNext, pc, sp, bit, sv,
					)
				}
			}

			// SP := SP-1
			spSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSUBStart+25)) +
				notHas(spID(sp-1))
			spNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseSUBStart+25))
			fmt.Fprintf(b, "%s .wz-sub-p%d-s%d-sp-set{visibility:visible;pointer-events:auto}\n", spSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-sub-p%d-s%d-sp-next{visibility:visible;pointer-events:auto}\n", spNextSel, pc, sp)

			// PC := PC+1
			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseSUBStart+26)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp-1)) +
				has(phaseID(phaseSUBStart+26))
			fmt.Fprintf(b, "%s .wz-sub-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-sub-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-sub-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseSUBEnd)),
	)
}
