package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type settingsRepositoryStub struct {
	activeQAConfigVersion QAConfigVersion
	activeQAConfigErr     error
	createdAgent          AgentConfig
	createCalled          bool
}

func (r *settingsRepositoryStub) GetActiveQAConfig(context.Context) (RetrievalSettings, []string, error) {
	return RetrievalSettings{TopK: 5, ScoreThreshold: .7, RerankThreshold: .5, RerankTopN: 3}.WithScoreThresholdConfigured(), []string{"kb-old"}, nil
}
func (r *settingsRepositoryStub) GetActiveQAConfigVersion(context.Context) (QAConfigVersion, error) {
	if r.activeQAConfigErr != nil {
		return QAConfigVersion{}, r.activeQAConfigErr
	}
	return r.activeQAConfigVersion, nil
}
func (r *settingsRepositoryStub) CreateQAConfigVersion(_ context.Context, _ string, _ RetrievalSettings, _ []string, agent AgentConfig) error {
	r.createCalled = true
	r.createdAgent = agent
	return nil
}
func (r *settingsRepositoryStub) GetActiveLLMConfig(context.Context) (StoredLLMConfig, error) {
	return StoredLLMConfig{
		Provider: "direct", APIEndpoint: "https://llm.example.test/v1", APIKeyLast4: "1234",
		TokenHeader: "Authorization", Model: "model", TimeoutSeconds: 30, Temperature: .7, MaxTokens: 1024,
	}, nil
}
func (r *settingsRepositoryStub) GetActiveLLMConfigVersion(context.Context) (LLMConfigVersion, error) {
	return LLMConfigVersion{}, nil
}
func (r *settingsRepositoryStub) CreateLLMConfigVersion(context.Context, string, StoredLLMConfig) error {
	return nil
}
func (r *settingsRepositoryStub) GetRuntimeSetting(context.Context, string) (string, error) {
	return "system prompt", nil
}
func (r *settingsRepositoryStub) UpsertRuntimeSetting(context.Context, string, string) error {
	return nil
}
func (r *settingsRepositoryStub) ListMCPServers(context.Context) ([]MCPServerRecord, error) {
	return nil, nil
}
func (r *settingsRepositoryStub) GetMCPServer(context.Context, string) (MCPServerRecord, error) {
	return MCPServerRecord{}, nil
}
func (r *settingsRepositoryStub) CreateMCPServer(context.Context, MCPServerRecord) (MCPServerRecord, error) {
	return MCPServerRecord{}, nil
}
func (r *settingsRepositoryStub) UpdateMCPServer(context.Context, MCPServerRecord) (MCPServerRecord, error) {
	return MCPServerRecord{}, nil
}
func (r *settingsRepositoryStub) DeleteMCPServer(context.Context, string) error {
	return nil
}
func (r *settingsRepositoryStub) UpdateMCPConnectionStatus(context.Context, string, int, *time.Time, string) error {
	return nil
}
func (r *settingsRepositoryStub) WriteAuditLog(context.Context, AuditLog) error {
	return nil
}

type passthroughCipher struct{}

func (passthroughCipher) Encrypt(value string) ([]byte, error) { return []byte(value), nil }
func (passthroughCipher) Decrypt(value []byte) (string, error) { return string(value), nil }

type noopLLMTester struct{}

func (noopLLMTester) TestLLM(context.Context, RuntimeLLMConfig) (LLMConnectionTestResult, error) {
	return LLMConnectionTestResult{Success: true}, nil
}

type noopMCPTester struct{}

func (noopMCPTester) TestMCP(context.Context, RuntimeMCPConfig) (MCPConnectionTestResult, error) {
	return MCPConnectionTestResult{Success: true}, nil
}

func TestUpdateSettingsPreservesActiveAgentConfig(t *testing.T) {
	repository := &settingsRepositoryStub{activeQAConfigVersion: QAConfigVersion{
		ID: "qa-config-id",
		Agent: AgentConfig{
			MaxIterations:         8,
			ToolTimeoutSeconds:    11,
			ModelTimeoutSeconds:   70,
			OverallTimeoutSeconds: 150,
			EnabledToolNames:      []string{"search_knowledge", "get_citation_source"},
		},
	}}
	settings, err := NewConfigService(repository, passthroughCipher{}, BootstrapSettings{}, noopLLMTester{}, noopMCPTester{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = settings.UpdateSettings(context.Background(), "user-1", "request-1", UpdateQASettingsInput{
		Retrieval: &RetrievalSettings{TopK: 6, ScoreThreshold: .6, RerankThreshold: .4, RerankTopN: 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !repository.createCalled {
		t.Fatal("CreateQAConfigVersion was not called")
	}
	if !reflect.DeepEqual(repository.createdAgent, repository.activeQAConfigVersion.Agent) {
		t.Fatalf("agent=%+v, want %+v", repository.createdAgent, repository.activeQAConfigVersion.Agent)
	}
}

func TestUpdateSettingsBootstrapsAgentConfigWhenActiveConfigMissing(t *testing.T) {
	repository := &settingsRepositoryStub{activeQAConfigErr: NewError(CodeNotFound, "QA configuration not found", errors.New("no rows"))}
	settings, err := NewConfigService(repository, passthroughCipher{}, BootstrapSettings{}, noopLLMTester{}, noopMCPTester{})
	if err != nil {
		t.Fatal(err)
	}

	ids := []string{"kb-new"}
	_, err = settings.UpdateSettings(context.Background(), "user-1", "request-1", UpdateQASettingsInput{DefaultKnowledgeBaseIDs: &ids})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(repository.createdAgent, DefaultAgentConfig()) {
		t.Fatalf("agent=%+v, want default %+v", repository.createdAgent, DefaultAgentConfig())
	}
}
