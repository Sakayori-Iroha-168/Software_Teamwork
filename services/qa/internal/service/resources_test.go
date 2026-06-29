package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCreateLLMConfigVersionRequiresAIGatewayProfile(t *testing.T) {
	repo := &fakeResourceRepository{}
	service, err := NewResourceService(repo, fakeKnowledgeRetriever{}, &fakeLLMTester{}, RuntimeLLMConfig{}, fakeRunCanceller{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.CreateLLMConfigVersion(context.Background(), "admin-user", CreateLLMConfigVersionInput{
		Provider: "direct", ModelName: "model", TimeoutSeconds: 30, MaxTokens: 1024,
	})

	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeValidation {
		t.Fatalf("err = %v, want validation error", err)
	}
	if repo.createdLLM {
		t.Fatal("repository should not be called for invalid provider/profile input")
	}
	if _, ok := appErr.Fields["provider"]; !ok {
		t.Fatalf("validation fields = %#v, want provider error", appErr.Fields)
	}
	if _, ok := appErr.Fields["profileId"]; !ok {
		t.Fatalf("validation fields = %#v, want profileId error", appErr.Fields)
	}
}

func TestLLMConnectionTestSanitizesProviderErrorAndSavesResult(t *testing.T) {
	repo := &fakeResourceRepository{}
	tester := fakeLLMTester{err: errors.New("raw provider failure: https://internal.example/key sk-secret full prompt text")}
	service, err := NewResourceService(repo, fakeKnowledgeRetriever{}, &tester, RuntimeLLMConfig{Endpoint: "https://gateway.example", Token: "gateway-token", TokenHeader: "Authorization"}, fakeRunCanceller{})
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.TestLLMConnection(context.Background(), "admin-user", LLMProfileTestInput{
		Provider: "ai-gateway", ProfileID: "profile-chat", ModelName: "qwen-test", TimeoutSeconds: 12,
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatalf("result.Success = true, want false")
	}
	if result.ErrorCode != "dependency_error" || result.ErrorMessage != "AI Gateway connection test failed" {
		t.Fatalf("result = %+v, want sanitized dependency error", result)
	}
	for _, forbidden := range []string{"internal.example", "sk-secret", "full prompt text"} {
		if strings.Contains(result.ErrorMessage, forbidden) {
			t.Fatalf("error message leaked %q: %s", forbidden, result.ErrorMessage)
		}
	}
	if repo.savedLLMTest.UserID != "admin-user" {
		t.Fatalf("saved user = %q, want admin-user", repo.savedLLMTest.UserID)
	}
	if repo.savedLLMTest.Result.ModelName != "qwen-test" || repo.savedLLMTest.Result.TestedAt.IsZero() {
		t.Fatalf("saved result = %+v", repo.savedLLMTest.Result)
	}
}

func TestLLMConnectionTestUsesAIGatewayProfileRuntime(t *testing.T) {
	tester := fakeLLMTester{result: LLMConnectionTestResult{Success: true, Model: "qwen-test", LatencyMS: 7}}
	service, err := NewResourceService(&fakeResourceRepository{}, fakeKnowledgeRetriever{}, &tester, RuntimeLLMConfig{Endpoint: "https://gateway.example", Token: "gateway-token", TokenHeader: "Authorization"}, fakeRunCanceller{})
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.TestLLMConnection(context.Background(), "admin-user", LLMProfileTestInput{
		Provider: "ai-gateway", ProfileID: "profile-chat", ModelName: "qwen-test", TimeoutSeconds: 12,
	})

	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result.Success = false, want true")
	}
	if tester.seen.ProfileID != "profile-chat" || tester.seen.Model != "qwen-test" || tester.seen.Timeout != 12*time.Second {
		t.Fatalf("tester runtime = %+v", tester.seen)
	}
}

func TestQAConfigVersionExposesDocumentedFlatAgentFields(t *testing.T) {
	version := QAConfigVersion{
		ID:                      "qa_cfg_1",
		VersionNo:               1,
		DefaultKnowledgeBaseIDs: []string{"kb-1"},
		KnowledgeBases:          []ConfigKnowledgeBase{{ID: "kb-1", SortOrder: 0}},
		Retrieval:               RetrievalSettings{TopK: 5, ScoreThreshold: 0.65},
		MaxIterations:           5,
		ToolTimeoutSeconds:      10,
		ModelTimeoutSeconds:     60,
		OverallTimeoutSeconds:   120,
		EnabledToolNames:        []string{"search_knowledge"},
		Agent: AgentConfig{
			MaxIterations:         5,
			ToolTimeoutSeconds:    10,
			ModelTimeoutSeconds:   60,
			OverallTimeoutSeconds: 120,
			EnabledToolNames:      []string{"search_knowledge"},
		},
		IsActive:  true,
		CreatedAt: time.Unix(0, 0).UTC(),
	}

	raw, err := json.Marshal(version)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"maxIterations", "toolTimeoutSeconds", "modelTimeoutSeconds", "overallTimeoutSeconds", "enabledToolNames", "agent", "knowledgeBases"} {
		if _, ok := payload[field]; !ok {
			t.Fatalf("QAConfigVersion JSON missing documented field %q: %s", field, string(raw))
		}
	}
}

type savedLLMTest struct {
	UserID string
	Result LLMProfileTestResult
}

type fakeResourceRepository struct {
	createdLLM   bool
	savedLLMTest savedLLMTest
}

func (f *fakeResourceRepository) GetResponseRun(context.Context, string, string) (ResponseRun, error) {
	return ResponseRun{}, nil
}
func (f *fakeResourceRepository) CancelResponseRun(context.Context, string, string) (ResponseRun, error) {
	return ResponseRun{}, nil
}
func (f *fakeResourceRepository) ListStreamEvents(context.Context, string, string, string, int) ([]StreamEvent, error) {
	return []StreamEvent{}, nil
}
func (f *fakeResourceRepository) ListMessageCitations(context.Context, string, string) ([]Citation, error) {
	return []Citation{}, nil
}
func (f *fakeResourceRepository) GetCitation(context.Context, string, string) (Citation, error) {
	return Citation{}, nil
}
func (f *fakeResourceRepository) LookupCitations(context.Context, string, []string) ([]Citation, error) {
	return []Citation{}, nil
}
func (f *fakeResourceRepository) ListToolCalls(context.Context, string, string) ([]AgentToolCall, error) {
	return []AgentToolCall{}, nil
}
func (f *fakeResourceRepository) GetActiveQAConfigVersion(context.Context) (QAConfigVersion, error) {
	return QAConfigVersion{}, nil
}
func (f *fakeResourceRepository) CreateQAConfigVersionResource(context.Context, string, CreateQAConfigVersionInput) (QAConfigVersion, error) {
	return QAConfigVersion{}, nil
}
func (f *fakeResourceRepository) GetActiveLLMConfigVersion(context.Context) (LLMConfigVersion, error) {
	return LLMConfigVersion{}, nil
}
func (f *fakeResourceRepository) CreateLLMConfigVersionResource(_ context.Context, _ string, input CreateLLMConfigVersionInput) (LLMConfigVersion, error) {
	f.createdLLM = true
	return LLMConfigVersion{Provider: input.Provider, ProfileID: input.ProfileID, ModelName: input.ModelName}, nil
}
func (f *fakeResourceRepository) SaveLLMConnectionTest(_ context.Context, userID string, result LLMProfileTestResult) (LLMProfileTestResult, error) {
	f.savedLLMTest = savedLLMTest{UserID: userID, Result: result}
	return result, nil
}
func (f *fakeResourceRepository) SaveRetrievalTestRun(context.Context, string, RetrievalTestInput, []RetrievalTestResult, time.Duration, error) (RetrievalTestRun, error) {
	return RetrievalTestRun{}, nil
}
func (f *fakeResourceRepository) GetRetrievalTestRun(context.Context, string, string) (RetrievalTestRun, error) {
	return RetrievalTestRun{}, nil
}
func (f *fakeResourceRepository) GetMetricsOverview(context.Context, int) (MetricsOverview, error) {
	return MetricsOverview{}, nil
}
func (f *fakeResourceRepository) GetMetricsTrend(context.Context, int) (MetricsTrend, error) {
	return MetricsTrend{}, nil
}
func (f *fakeResourceRepository) GetTopQueries(context.Context, int, int) ([]TopQuery, error) {
	return []TopQuery{}, nil
}
func (f *fakeResourceRepository) GetIntentDistribution(context.Context, int) ([]IntentDistribution, error) {
	return []IntentDistribution{}, nil
}

type fakeKnowledgeRetriever struct{}

func (fakeKnowledgeRetriever) Retrieve(context.Context, string, RetrievalTestInput) ([]RetrievalTestResult, error) {
	return []RetrievalTestResult{}, nil
}

type fakeLLMTester struct {
	result LLMConnectionTestResult
	err    error
	seen   RuntimeLLMConfig
}

func (f *fakeLLMTester) TestLLM(_ context.Context, config RuntimeLLMConfig) (LLMConnectionTestResult, error) {
	f.seen = config
	if f.err != nil {
		return LLMConnectionTestResult{}, f.err
	}
	if f.result.Model == "" {
		f.result = LLMConnectionTestResult{Success: true, Model: config.Model, LatencyMS: 1}
	}
	return f.result, nil
}

type fakeRunCanceller struct{}

func (fakeRunCanceller) CancelActiveRun(string) {}
