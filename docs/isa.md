# Runtime8 ISA Reference

Runtime8 uses a minimal 8-bit stack ISA encoded in 16-bit words:

- word format: `[opcode:8 | operand:8]`
- value domain: unsigned 8-bit (`0..255`)
- stack machine bounds: depth 8

## Instructions

### `HLT`
- Opcode: `0x00`
- Operand: none
- Effect: halt execution

### `PSH imm8`
- Opcode: `0x01`
- Operand: `imm8`
- Stack effect: `-- imm8`
- Literal forms: decimal (`42`), hex (`0x2A`), binary (`0b00101010`)

### `POP`
- Opcode: `0x02`
- Operand: none
- Stack effect: `a --`
- Semantics: drops top of stack

### `ADD`
- Opcode: `0x04`
- Operand: none
- Stack effect: `a b -- (a+b mod 256)`
- Semantics: unsigned 8-bit wrap (carry-out discarded)

### `SUB`
- Opcode: `0x03`
- Operand: none
- Stack effect: `a b -- (a-b mod 256)`
- Semantics: unsigned 8-bit wrap (borrow-out discarded)

### `AND`
- Opcode: `0x05`
- Operand: none
- Stack effect: `a b -- (a&b)`
- Semantics: bitwise AND on 8-bit cells

### `XOR`
- Opcode: `0x06`
- Operand: none
- Stack effect: `a b -- (a^b)`
- Semantics: bitwise XOR on 8-bit cells

### `OR`
- Opcode: `0x07`
- Operand: none
- Stack effect: `a b -- (a|b)`
- Semantics: bitwise OR on 8-bit cells

### `DUP`
- Opcode: `0x0E`
- Operand: none
- Stack effect: `a -- a a`
- Semantics: duplicates top of stack

### `SWP`
- Opcode: `0x0F`
- Operand: none
- Stack effect: `a b -- b a`
- Semantics: swaps top two stack cells

### `OVR`
- Opcode: `0x10`
- Operand: none
- Stack effect: `a b -- a b a`
- Semantics: copies second-to-top cell to top

### `INC`
- Opcode: `0x11`
- Operand: none
- Stack effect: `a -- (a+1 mod 256)`
- Semantics: unsigned 8-bit wrap increment

### `DEC`
- Opcode: `0x12`
- Operand: none
- Stack effect: `a -- (a-1 mod 256)`
- Semantics: unsigned 8-bit wrap decrement

### `CLL imm`
- Opcode: `0x13`
- Operand: `imm`
- Stack effect: `-- ret` (pushes `PC+1` as return target)
- Semantics: push return target then set `PC = imm`

### `RET`
- Opcode: `0x14`
- Operand: none
- Stack effect: `ret --`
- Semantics: pops top-of-stack into `PC`

### `NOT`
- Opcode: `0x09`
- Operand: none
- Stack effect: `a -- (~a)`
- Semantics: bitwise NOT on 8-bit cells (high bits masked by 8-bit storage)

### `SHL`
- Opcode: `0x0A`
- Operand: none
- Stack effect: `a -- ((a<<1) mod 256)`
- Semantics: logical left shift by 1, low bit becomes 0

### `SHR`
- Opcode: `0x0B`
- Operand: none
- Stack effect: `a -- (a>>1)`
- Semantics: logical right shift by 1, high bit becomes 0

### `JMP imm`
- Opcode: `0x0C`
- Operand: `imm`
- Stack effect: unchanged
- Semantics: absolute `PC = imm` (`0..31`)

### `JNZ imm`
- Opcode: `0x0D`
- Operand: `imm`
- Stack effect: `a -- a` (non-destructive condition check on TOS)
- Semantics: if `a != 0` then `PC = imm`, else `PC = PC+1`

### `SAY`
- Opcode: `0x08`
- Operand: none
- Stack effect: `a -- a`
- Effect: copies TOS to output latch (non-destructive)

## Example

```asm
PSH 2
PSH 3
ADD
SAY
HLT
```
