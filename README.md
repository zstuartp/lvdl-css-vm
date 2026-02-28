# LVDL-VM

A small 8-bit stack machine that executes in HTML+CSS only.

The current runtime is intentionally minimal:
- Runtime execution is done by CSS selectors and radio state.
- A tiny JS clock is included for auto-stepping; it only clicks the single
  currently visible write-zone label.
- Clock presets are fixed: `SLOW` (500ms), `MEDIUM` (250ms), `FAST` (100ms),
  `TURBO` (10ms), `MAX` (0ms).
- There is no JS interpreter/emulator logic, no in-browser editor, and no
  pre-trace mode.
- The ALU path for `ADD` is a runtime bitwise ripple-carry chain in CSS.
- Bitwise `AND`/`XOR`/`OR`/`NOT`/`SHL`/`SHR` are computed at runtime from stack-bit radios.
- Control flow `JMP`/`JNZ` is runtime PC-routing in CSS.

> Requires a modern browser with `:has()` support.

## What It Implements

- Word format: 16-bit instruction words (`[opcode:8 | operand:8]`)
- Stack cells: 8-bit unsigned values
- Fixed machine bounds: program 32 words, stack depth 8
- Instruction set: `HLT`, `PSH`, `POP`, `ADD`, `SUB`, `AND`, `XOR`, `OR`, `DUP`, `SWP`, `OVR`, `INC`, `DEC`, `CLL`, `RET`, `NOT`, `SHL`, `SHR`, `JMP`, `JNZ`, `SAY`
- `ADD` semantics: unsigned wrap modulo 256
- `SUB` semantics: unsigned wrap modulo 256 (`a-b mod 256`)
- `AND` semantics: bitwise `a & b`
- `XOR` semantics: bitwise `a ^ b`
- `OR` semantics: bitwise `a | b`
- `DUP` semantics: duplicate top of stack (`x -> x x`)
- `SWP` semantics: swap top two stack cells (`a b -> b a`)
- `OVR` semantics: copy second-to-top to top (`a b -> a b a`)
- `INC` semantics: increment top of stack (`a -> a+1 mod 256`)
- `DEC` semantics: decrement top of stack (`a -> a-1 mod 256`)
- `CLL` semantics: push return `PC+1`, then jump to `imm`
- `RET` semantics: pop return target into `PC`
- `POP` semantics: drop top of stack (`x ->`)
- `NOT` semantics: bitwise `~a` on 8-bit cells
- `SHL` semantics: logical left shift `(a << 1) mod 256`
- `SHR` semantics: logical right shift `(a >> 1)`
- `JMP` semantics: absolute jump to `imm` (`0..31`)
- `JNZ` semantics: if `TOS != 0` then `PC = imm`, else `PC = PC+1`
- `PSH` accepts `imm8` in decimal, hex (`0x..`), or binary (`0b..`)
- `JMP`/`JNZ` targets accept `imm8` or labels (`label:` / `JMP label`)

## Quick Start

```bash
go build ./cmd/lvdlc
./lvdlc -spec examples/standard.lvdl -asm examples/add_demo.asm -profile runtime8 -mode live-pure-css -out vm-css-pure.html
./lvdlc -spec examples/standard.lvdl -asm examples/add_demo.asm -profile runtime8 -mode live-js-clock -out vm-js-clock.html
./lvdlc -spec examples/standard.lvdl -asm examples/add_demo.asm -profile runtime8 -mode live-pure-css -opt none -out vm-css-unpruned.html
open vm-js-clock.html
```

Click the single visible write-zone label to advance one micro-step.
In `*-js-clock.html`, use `Run`/`Stop` to auto-step.

## Acceptance Demos

- `examples/add_demo.asm` (`2 + 3 => 5`)
- `examples/overflow_demo.asm` (`250 + 10 => 4`)
- `examples/sub_demo.asm` (`7 - 2 => 5`)
- `examples/and_demo.asm` (`0b10101010 & 0b11001100 => 136`)
- `examples/xor_demo.asm` (`0b10101010 ^ 0b11001100 => 102`)
- `examples/or_demo.asm` (`0b10100000 | 0b00001111 => 175`)
- `examples/dup_demo.asm` (`42 DUP ADD => 84`)
- `examples/swp_demo.asm` (`7 2 SWP SUB => 251`)
- `examples/ovr_demo.asm` (`4 3 OVR ADD ADD => 11`)
- `examples/inc_demo.asm` (`255 INC => 0`)
- `examples/dec_demo.asm` (`0 DEC => 255`)
- `examples/cll_ret_demo.asm` (`CLL` to subroutine then `RET`, outputs `7`)
- `examples/pop_demo.asm` (`9 POP 5 => 5`)
- `examples/not_demo.asm` (`~0b01010101 => 170`)
- `examples/shl_demo.asm` (`0b10101010 << 1 => 84`)
- `examples/shr_demo.asm` (`0b10101010 >> 1 => 85`)
- `examples/jmp_demo.asm` (skips one instruction, outputs `1`)
- `examples/jnz_demo.asm` (both JNZ paths in one program, outputs `3`)
- `examples/isa_smoke.asm` (runs all ISA ops once, outputs `80`)

Generate committed example HTML files:

```bash
make examples
```

This writes both:
- `*-css-pure.html` (no script, manual stepping)
- `*-js-clock.html` (click-only JS clock)

Generate tiny-profile examples:

```bash
make examples-tiny
```

## JS Clock Contract

In `*-js-clock.html`, JavaScript only does scheduling and button wiring:

- finds visible write-zone labels in `.write-bus`
- requires exactly one visible label
- advances by calling `.click()` on that label
- reports `halted` on zero labels, `conflict` on multiple labels

It does not compute transitions and does not write radio `.checked` state.

## CLI

`lvdlc` supports two output modes:

- `-mode live-pure-css`
- `-mode live-js-clock`

`lvdlc` profile presets:

- `-profile runtime8` (default)
- `-profile runtime8-tiny` (16-word program space; expects `PC` width 4 in spec)

Tiny profile reference files:

- `examples/standard-tiny.lvdl`
- `examples/tiny_add_demo.asm`
- `examples/tiny_jnz_demo.asm`

`lvdlc` optimization modes:

- `-opt control-state` (default): prune unreachable `(pc,sp)` control paths
  and trim rendered stack/program UI to reachable bounds
- `-opt none`: emit full per-instruction/per-SP control paths

## Size Report

Compare output size with and without compile-time pruning:

```bash
make size-report
```

Enforce key artifact size budgets:

```bash
make size-check
```

Run the full publication gate:

```bash
make prepublish
```

## Project Layout

```text
cmd/lvdlc/       compiler entry point
examples/        standard spec + ISA demo programs
internal/
  asm/           16-bit assembler ([opcode:8|operand:8])
  emit/          CSS runtime emitter + click-only JS clock (no pre-trace)
  lvdl/          .lvdl parser/decoder
```

## Documentation

- `docs/architecture.md`

## License

Apache 2.0. See `LICENSE`.
