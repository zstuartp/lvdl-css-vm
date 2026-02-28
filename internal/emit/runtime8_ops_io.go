// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import (
	"fmt"
	"strings"
)

func genSAYCSS(b *strings.Builder, words []decodedWord, plan runtimePlan) {
	for pc := 0; pc < runtimeProgWords; pc++ {
		if words[pc].Name != "SAY" {
			continue
		}
		nextPC := pc + 1

		for sp := 1; sp <= runtimeStackSize; sp++ {
			if !plan.active(pc, sp) {
				continue
			}
			top := sp - 1
			for bit := 0; bit < runtimeWordBits; bit++ {
				ph := phaseSAYStart + bit
				for sv := 0; sv <= 1; sv++ {
					base := "body" +
						has(opcID(pc)) +
						has(pcID(pc)) +
						has(ospID(sp)) +
						has(spID(sp)) +
						has(phaseID(ph)) +
						has(stkBitID(top, bit, sv))

					setSel := base + notHas(outBitID(bit, sv))
					nextSel := base + has(outBitID(bit, sv))
					fmt.Fprintf(b, "%s .wz-say-p%d-s%d-b%d-set%d{visibility:visible;pointer-events:auto}\n", setSel, pc, sp, bit, sv)
					fmt.Fprintf(b, "%s .wz-say-p%d-s%d-b%d-next%d{visibility:visible;pointer-events:auto}\n", nextSel, pc, sp, bit, sv)
				}
			}

			validSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSAYStart+8)) +
				notHas(outValidID(1))
			validNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSAYStart+8)) +
				has(outValidID(1))
			fmt.Fprintf(b, "%s .wz-say-p%d-s%d-valid-set{visibility:visible;pointer-events:auto}\n", validSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-say-p%d-s%d-valid-next{visibility:visible;pointer-events:auto}\n", validNextSel, pc, sp)

			pcSetSel := "body" +
				has(opcID(pc)) +
				has(pcID(pc)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSAYStart+9)) +
				notHas(pcID(nextPC))
			pcNextSel := "body" +
				has(opcID(pc)) +
				has(pcID(nextPC)) +
				has(ospID(sp)) +
				has(spID(sp)) +
				has(phaseID(phaseSAYStart+9))
			fmt.Fprintf(b, "%s .wz-say-p%d-s%d-pc-set{visibility:visible;pointer-events:auto}\n", pcSetSel, pc, sp)
			fmt.Fprintf(b, "%s .wz-say-p%d-s%d-pc-next{visibility:visible;pointer-events:auto}\n", pcNextSel, pc, sp)
		}
	}

	fmt.Fprintf(
		b,
		"body%s .wz-say-reset{visibility:visible;pointer-events:auto}\n",
		has(phaseID(phaseSAYEnd)),
	)
}
