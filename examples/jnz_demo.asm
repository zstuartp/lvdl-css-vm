  PSH 0
  JNZ take_first
  PSH 1
  JMP after_first
take_first:
  PSH 99
after_first:
  PSH 1
  JNZ take_second
  PSH 99
  JMP after_second
take_second:
  PSH 2
after_second:
  ADD
  SAY
  HLT
