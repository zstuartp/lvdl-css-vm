  CLL fn
  PSH 2
  PSH 3
  ADD
  PSH 0b11110000
  AND
  PSH 0b10101010
  XOR
  PSH 0b00001111
  OR
  NOT
  SHL
  SHR
  INC
  DEC
  DUP
  SWP
  OVR
  POP
  AND
  JMP chk_nonzero
  PSH 99
chk_nonzero:
  JNZ after_skip88
  PSH 88
after_skip88:
  PSH 0
  SUB
  SAY
  HLT
fn:
  RET
