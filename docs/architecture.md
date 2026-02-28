# LVDL-VM Runtime8 Architecture

Runtime8 uses two HTML outputs:

- `*-css-pure.html` from `live-pure-css` (manual clicks only)
- `*-js-clock.html` from `live-js-clock` (auto-step clock)
- no precomputed transition graph
- no JS in `*-css-pure.html`

Each micro-step is still one radio write. In clock mode, JS only clicks the
single currently visible write-zone label.

The emitter supports two optimization modes:

- `control-state` (default): prunes unreachable `(pc,sp)` paths
- `none`: emits full per-instruction/per-SP paths

## Machine Model

- Program memory: 32 words (`runtime8`) or 16 words (`runtime8-tiny`)
- Word format: `[opcode:8 | operand:8]`
- Stack depth: 8
- Stack cell width: 8 bits (unsigned)
- ISA: `HLT`, `PSH`, `POP`, `ADD`, `SUB`, `AND`, `XOR`, `OR`, `DUP`, `SWP`, `OVR`, `INC`, `DEC`, `CLL`, `RET`, `NOT`, `SHL`, `SHR`, `JMP`, `JNZ`, `SAY`

## State Encoding

All machine state is one-hot radio groups using compact IDs:

- PC: `p0..p31`
- SP: `s0..s8`
- Stack bits: `k{cell}{bit}{0|1}`
- ALU staging bits: `u{bit}{0|1}`
- Carry chain bits: `c{0..8}{0|1}`
- Output bits: `o{bit}{0|1}` and `ov{0|1}`
- Micro-phase: `h0..h382`

## Control Flow

At `h0`, CSS dispatches by current `PC`:

- `PSH` -> phase range `h1..h11`
- `ADD` -> phase range `h20..h47`
- `DUP` -> phase range `h48..h59`
- `SWP` -> phase range `h250..h275`
- `OVR` -> phase range `h280..h290`
- `INC` -> phase range `h300..h326`
- `DEC` -> phase range `h330..h356`
- `CLL` -> phase range `h360..h370`
- `RET` -> phase range `h380..h382`
- `SUB` -> phase range `h220..h247`
- `SAY` -> phase range `h60..h70`
- `POP` -> phase range `h71..h73`
- `AND` -> phase range `h80..h98`
- `XOR` -> phase range `h100..h118`
- `OR` -> phase range `h120..h138`
- `NOT` -> phase range `h140..h157`
- `SHL` -> phase range `h160..h177`
- `SHR` -> phase range `h180..h197`
- `JMP` -> phase range `h200..h201`
- `JNZ` -> phase range `h210..h211`
- `HLT` -> no visible write-zone (halted)

Each phase has two selector outcomes:

1. `set` label: writes target radio when target is not yet correct.
2. `next` label: advances phase when target is already correct.

This gives deterministic forward progress with one radio write per click.

## Compile-Time Pruning

In `control-state` mode, the compiler runs an optimization pipeline:

1. control reachability from `(pc=0,sp=0)` over `(pc,sp)` pairs
2. stack UI trim to max reachable `SP`
3. program UI trim to max reachable `PC`

The pass is control-only (no value tracing). For value-dependent branches
(`JNZ`), both control edges are kept.

## ADD Datapath (Runtime Bitwise)

For `SP = s`:

- Inputs: `A = k[s-2]`, `B = k[s-1]`
- Destination: `k[s-2]`

Sequence:

1. Force `car0 = 0`
2. For each bit `i = 0..7`:
   - `sum_i = A_i XOR B_i XOR car_i`
   - `car_{i+1} = (A_i & B_i) | (A_i & car_i) | (B_i & car_i)`
3. Write `sum_0..sum_7` into destination bits
4. Set `SP = s-1`
5. Set `PC = PC+1`
6. Reset phase to `h0`

Overflow carry (`car8`) is discarded, so arithmetic is modulo 256.

## SUB Datapath (Runtime Bitwise)

For `SP = s`:

- Inputs: `A = k[s-2]`, `B = k[s-1]`
- Destination: `k[s-2]`

Sequence:

1. Force `borrow0 = 0`
2. For each bit `i = 0..7`:
   - `diff_i = A_i XOR B_i XOR borrow_i`
   - `borrow_{i+1} = 1` iff `(A_i - B_i - borrow_i) < 0`
3. Write `diff_0..diff_7` into destination bits
4. Set `SP = s-1`
5. Set `PC = PC+1`
6. Reset phase to `h0`

Final borrow is discarded, so arithmetic is modulo 256.

## Bitwise Datapaths

`AND`, `XOR`, `OR`, `NOT`, `SHL`, and `SHR` use a per-bit combinational selector path:

1. For each bit `i = 0..7`, compute:
   - `u_i = A_i & B_i` for `AND`
   - `u_i = A_i ^ B_i` for `XOR`
   - `u_i = A_i | B_i` for `OR`
   - `u_i = ~A_i` for `NOT`
   - `u_0 = 0` and `u_i = A_{i-1}` for `SHL`
   - `u_i = A_{i+1}` for `i=0..6`, `u_7 = 0` for `SHR`
2. Write `u_0..u_7` to destination (`k[s-2]` for binary ops, `k[s-1]` for `NOT`/`SHL`/`SHR`)
3. For `AND`/`XOR`/`OR`: set `SP = s-1` (`NOT`/`SHL`/`SHR` leave `SP` unchanged)
4. Set `PC = PC+1`
5. Reset phase to `h0`

## DUP Datapath

`DUP` copies top-of-stack to a new top cell:

1. For each bit `i = 0..7`, copy `stk[SP-1].i -> stk[SP].i`
2. Set `SP = SP+1`
3. Set `PC = PC+1`
4. Reset phase to `h0`

## POP Datapath

`POP` drops top-of-stack:

1. Set `SP = SP-1`
2. Set `PC = PC+1`
3. Reset phase to `h0`

## SWP Datapath

`SWP` exchanges the top two stack cells:

1. Stage `stk[SP-1]` into `u0..u7`
2. Copy `stk[SP-2] -> stk[SP-1]`
3. Copy staged `u0..u7 -> stk[SP-2]`
4. Set `PC = PC+1`
5. Reset phase to `h0`

`SP` is unchanged.

## OVR Datapath

`OVR` duplicates second-to-top to a new top cell:

1. Copy `stk[SP-2] -> stk[SP]`
2. Set `SP = SP+1`
3. Set `PC = PC+1`
4. Reset phase to `h0`

## INC Datapath

`INC` increments top-of-stack with modulo-256 wrap:

1. Force `car0 = 1`
2. For each bit `i = 0..7`:
   - `u_i = A_i XOR car_i`
   - `car_{i+1} = A_i & car_i`
3. Write `u_0..u_7` back to `stk[SP-1]`
4. Set `PC = PC+1`
5. Reset phase to `h0`

`SP` is unchanged.

## DEC Datapath

`DEC` decrements top-of-stack with modulo-256 wrap:

1. Force `borrow0 = 1`
2. For each bit `i = 0..7`:
   - `u_i = A_i XOR borrow_i`
   - `borrow_{i+1} = (!A_i) & borrow_i`
3. Write `u_0..u_7` back to `stk[SP-1]`
4. Set `PC = PC+1`
5. Reset phase to `h0`

`SP` is unchanged.

## CLL Datapath

`CLL imm` uses the data stack for return targets:

1. Write return value `PC+1` into `stk[SP]` bits
2. Set `SP = SP+1`
3. Set `PC = imm`
4. Reset phase to `h0`

## RET Datapath

`RET` consumes the return target from top-of-stack:

1. Decode `stk[SP-1]` (0..program-max) into one-hot `PC`
2. Set `SP = SP-1`
3. Reset phase to `h0`

## Jump Datapath

`JMP imm` performs an absolute control transfer:

1. Set `PC = imm` (`0..31`)
2. Reset phase to `h0`

`SP` and stack bits are unchanged.

`JNZ imm` branches on top-of-stack:

1. If `TOS != 0`, set `PC = imm`
2. Else set `PC = PC+1`
3. Reset phase to `h0`

`JNZ` is non-destructive (`SP` and stack bits unchanged).

## Output

`SAY` copies top-of-stack bits to output bits and sets `ov1`.
A CSS-decoded 0..255 view is shown in the UI.

## Clock Invariant

Generated HTML includes a small JS clock for auto-stepping.
The clock does not compute transitions and does not write radio state.
It only clicks the currently visible write-zone label.
CSS selectors and radio state remain the execution engine.

Clock controls are fixed presets:

- `SLOW` 500ms
- `MEDIUM` 250ms
- `FAST` 100ms
- `TURBO` 10ms
- `MAX` 0ms
