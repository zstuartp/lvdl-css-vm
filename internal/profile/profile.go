// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

// Package profile defines machine-profile presets used by compiler/emitter.
package profile

import "fmt"

// Machine describes one runtime profile preset.
type Machine struct {
	Name         string
	ProgramWords int
	StackWords   int
	StackBits    int
	WordBits     int
	OpcodeBits   int
	OperandBits  int
	PCBits       int
	SPBits       int
}

// MaxProgramPC returns the highest valid PC value for this profile.
func (m Machine) MaxProgramPC() int {
	if m.ProgramWords <= 0 {
		return -1
	}
	return m.ProgramWords - 1
}

// Runtime8 is the default profile for this branch.
var Runtime8 = Machine{
	Name:         "runtime8",
	ProgramWords: 32,
	StackWords:   8,
	StackBits:    8,
	WordBits:     16,
	OpcodeBits:   8,
	OperandBits:  8,
	PCBits:       5,
	SPBits:       4,
}

// Runtime8Tiny keeps the same datapath but smaller program space.
var Runtime8Tiny = Machine{
	Name:         "runtime8-tiny",
	ProgramWords: 16,
	StackWords:   8,
	StackBits:    8,
	WordBits:     16,
	OpcodeBits:   8,
	OperandBits:  8,
	PCBits:       4,
	SPBits:       4,
}

var byName = map[string]Machine{
	Runtime8.Name:     Runtime8,
	Runtime8Tiny.Name: Runtime8Tiny,
}

// Names returns the set of supported profile names.
func Names() []string {
	return []string{Runtime8.Name, Runtime8Tiny.Name}
}

// Resolve maps a profile name to a preset.
func Resolve(name string) (Machine, error) {
	m, ok := byName[name]
	if !ok {
		return Machine{}, fmt.Errorf(
			"unknown profile %q (supported: %s, %s)",
			name, Runtime8.Name, Runtime8Tiny.Name,
		)
	}
	return m, nil
}
