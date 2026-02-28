// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import "lvdl-vm/internal/profile"

// OptimizationMode controls compile-time optimizations for generated runtime CSS.
type OptimizationMode string

const (
	// OptimizationNone emits all valid per-instruction/per-SP control paths.
	OptimizationNone OptimizationMode = "none"
	// OptimizationControlState prunes per-instruction/per-SP paths that are
	// unreachable from the initial (pc=0,sp=0) control state.
	OptimizationControlState OptimizationMode = "control-state"
)

// Options controls runtime HTML generation behavior.
type Options struct {
	Optimization OptimizationMode
	Profile      profile.Machine
}

func normalizedOptions(opts Options) Options {
	if opts.Optimization == "" {
		opts.Optimization = OptimizationControlState
	}
	if opts.Profile.Name == "" {
		opts.Profile = profile.Runtime8
	}
	return opts
}
