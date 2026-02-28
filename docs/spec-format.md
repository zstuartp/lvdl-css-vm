# Spec Format (Runtime8)

`.lvdl` files are parsed into machine and ISA metadata.

For the `runtime8` profile, the compiler validates and expects:

- `(word 16)`
- `(reg PC 5)`
- `(reg SP 4)`
- `(encoding (opcode-bits 8) (operand-bits 8))`
- first stack is 8 words x 8 bits
- ISA includes `HLT`, `PSH`, `ADD`, `SUB`, `AND`, `XOR`, `OR`, `NOT`, `SHL`, `SHR`, `JMP`, `JNZ`, `SAY`

## Minimal Example

```lisp
(lvdl
    (machine
        (word 16)
        (reg PC 5)
        (reg SP 4)
        (stack STK 8 8))
    (isa
        (encoding (opcode-bits 8) (operand-bits 8))
        (instr HLT 0x00 () ())
        (instr PSH 0x01 (imm8) ((push imm)))
        (instr ADD 0x04 () ((pop b) (pop a) (push (+ a b))))
        (instr SUB 0x03 () ((pop b) (pop a) (push (- a b))))
        (instr AND 0x05 () ((pop b) (pop a) (push (& a b))))
        (instr XOR 0x06 () ((pop b) (pop a) (push (^ a b))))
        (instr OR 0x07 () ((pop b) (pop a) (push (| a b))))
        (instr NOT 0x09 () ((pop a) (push (~ a))))
        (instr SHL 0x0A () ((pop a) (push (<< a 1))))
        (instr SHR 0x0B () ((pop a) (push (>> a 1))))
        (instr JMP 0x0C (addr) ((set PC addr)))
        (instr JNZ 0x0D (addr) ((if (!= TOS 0) (set PC addr) (set PC (+ PC 1)))))
        (instr SAY 0x08 () ((output TOS)))))
```

## Notes

- The parser itself is generic, but `cmd/lvdlc` enforces the runtime8 profile above.
- Micro-op lists in `.lvdl` are informational; execution semantics are in the emitter.
