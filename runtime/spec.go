package runtime

import (
	"fmt"
)

// Capability describes a high-level feature a runtime executor can enable.
type Capability string

const (
	CapabilityTools  Capability = "tools"
	CapabilityMemory Capability = "memory"
	CapabilityStream Capability = "stream"
)

// AgentSpec captures the immutable configuration for building runtime executors.
type AgentSpec struct {
	Name           string
	SystemPrompt   string
	Description    string
	MaxIterations  int
	Temperature    float64
	Capabilities   []Capability
	DefaultTools   []string
	DefaultMemory  []string
	SupportsStream bool
}

// Validate ensures the spec is well formed before building an executor.
func (s AgentSpec) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("runtime: agent spec name is required")
	}
	if s.SystemPrompt == "" {
		return fmt.Errorf("runtime: agent spec system prompt is required")
	}
	if s.MaxIterations <= 0 {
		return fmt.Errorf("runtime: max iterations must be positive")
	}
	if s.Temperature < 0 || s.Temperature > 2 {
		return fmt.Errorf("runtime: temperature must be between 0 and 2")
	}
	return nil
}

// HasCapability reports whether the spec declares the provided capability.
func (s AgentSpec) HasCapability(cap Capability) bool {
	for _, c := range s.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}
