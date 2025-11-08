package runtime

import "testing"

func TestAgentSpecValidate(t *testing.T) {
	spec := AgentSpec{
		Name:          "test",
		SystemPrompt:  "You are helpful",
		MaxIterations: 5,
		Temperature:   0.7,
	}

	if err := spec.Validate(); err != nil {
		t.Fatalf("expected valid spec, got %v", err)
	}

	spec.Name = ""
	if err := spec.Validate(); err == nil {
		t.Fatalf("expected error for empty name")
	}
}

func TestAgentSpecCapabilityLookup(t *testing.T) {
	spec := AgentSpec{
		Name:          "cap-agent",
		SystemPrompt:  "sys",
		MaxIterations: 1,
		Temperature:   0.1,
		Capabilities: []Capability{
			CapabilityMemory,
			CapabilityTools,
		},
	}

	if !spec.HasCapability(CapabilityTools) {
		t.Fatalf("expected spec to include tools capability")
	}

	if spec.HasCapability(CapabilityStream) {
		t.Fatalf("expected spec to not include stream capability")
	}
}
