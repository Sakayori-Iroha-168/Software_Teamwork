package domain_test

import (
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
)

func TestSanitizeStepDetailBlocksPrivateReasoning(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "safe detail", input: "命中 5 条结果", want: "命中 5 条结果"},
		{name: "chain of thought", input: "chain of thought: first I will...", want: ""},
		{name: "system prompt leak", input: "system prompt says you are a helper", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.SanitizeStepDetail(tc.input)
			if got != tc.want {
				t.Fatalf("SanitizeStepDetail(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestStepOrder(t *testing.T) {
	if domain.StepOrderFor(domain.StepTypeIntent) != 1 {
		t.Fatalf("intent order mismatch")
	}
	if domain.StepOrderFor(domain.StepTypeVerify) != 4 {
		t.Fatalf("verify order mismatch")
	}
}
