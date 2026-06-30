package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
)

type fakeRepository struct {
	conversation Conversation
	messages     []Message
	savedSteps   []ReasoningStep
	savedEvents  []StreamEvent
	run          ResponseRun
}

func (r *fakeRepository) CreateConversation(_ context.Context, value Conversation) (Conversation, error) {
	r.conversation = value
	return value, nil
}
func (r *fakeRepository) ListConversations(_ context.Context, _ string, options ConversationListOptions) (Page[Conversation], error) {
	return Page[Conversation]{Items: []Conversation{r.conversation}, Page: options.Page, PageSize: options.PageSize, Total: 1}, nil
}
func (r *fakeRepository) GetConversation(context.Context, string, string) (Conversation, error) {
	return r.conversation, nil
}
func (r *fakeRepository) UpdateConversation(_ context.Context, _ string, value Conversation) (Conversation, error) {
	r.conversation = value
	return value, nil
}
func (*fakeRepository) DeleteConversation(context.Context, string, string) error { return nil }
func (r *fakeRepository) ListMessages(context.Context, string, string, int, int) (Page[Message], error) {
	return Page[Message]{Items: append([]Message(nil), r.messages...), Page: 1, PageSize: 100, Total: len(r.messages)}, nil
}
func (r *fakeRepository) AppendMessages(_ context.Context, _, sessionID string, values ...Message) (ResponseRun, error) {
	r.messages = append(r.messages, values...)
	r.run = ResponseRun{ID: "run-id", SessionID: sessionID, UserMessageID: values[0].ID, AssistantMessageID: values[1].ID, Status: "running", MaxIterations: 5, CreatedAt: values[0].CreatedAt}
	return r.run, nil
}
func (r *fakeRepository) SaveStreamEvents(_ context.Context, _, _ string, events []StreamEvent) error {
	r.savedEvents = append([]StreamEvent(nil), events...)
	return nil
}
func (r *fakeRepository) GetResponseRun(context.Context, string, string) (ResponseRun, error) {
	r.run.Status = "completed"
	return r.run, nil
}
func (r *fakeRepository) UpdateMessage(_ context.Context, _ string, value Message) error {
	for index := range r.messages {
		if r.messages[index].ID == value.ID {
			r.messages[index] = value
			return nil
		}
	}
	return errors.New("message not found")
}
func (r *fakeRepository) SaveReasoningSteps(_ context.Context, _, _ string, steps []ReasoningStep) error {
	r.savedSteps = append([]ReasoningStep(nil), steps...)
	return nil
}

type fakeAgentRunner struct {
	input []agent.Message
}
type blockingAgentRunner struct{ started chan struct{} }

func (r blockingAgentRunner) RunWithObserver(ctx context.Context, _ []agent.Message, _ agent.Observer) (agent.Result, error) {
	close(r.started)
	<-ctx.Done()
	return agent.Result{}, ctx.Err()
}

func (r *fakeAgentRunner) RunWithObserver(_ context.Context, input []agent.Message, observer agent.Observer) (agent.Result, error) {
	r.input = append([]agent.Message(nil), input...)
	observer(agent.Event{Type: agent.EventModelStarted, Iteration: 1})
	observer(agent.Event{Type: agent.EventModelCompleted, Iteration: 1})
	final := agent.Message{Role: agent.RoleAssistant, Content: "测试回答"}
	return agent.Result{Final: final, Messages: append(input, final), Iterations: 1}, nil
}

type fakeRuntimeProvider struct {
	runner AgentRunner
	prompt string
}

func (p fakeRuntimeProvider) Acquire() (RuntimeSnapshot, func(), error) {
	return RuntimeSnapshot{Runner: p.runner, SystemPrompt: p.prompt}, func() {}, nil
}

func TestAskPersistsConversationMessagesAndDisplayableSteps(t *testing.T) {
	now := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", OwnerUserID: "user-id", Title: "新对话", Status: "active", CreatedAt: now, UpdatedAt: now}}
	runner := &fakeAgentRunner{}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: runner, prompt: "system prompt"})
	if err != nil {
		t.Fatal(err)
	}
	qa.now = func() time.Time { return now }
	var events []ProgressEvent
	result, err := qa.Ask(context.Background(), "user-id", "conversation-id", AskInput{Message: "锅炉检查要求", Mode: "knowledge_qa"}, func(event ProgressEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AssistantMessage.Content != "测试回答" || result.AssistantMessage.Status != "completed" {
		t.Fatalf("unexpected answer: %+v", result.AssistantMessage)
	}
	if repository.conversation.Title != "锅炉检查要求" {
		t.Fatalf("automatic title = %q", repository.conversation.Title)
	}
	if len(repository.messages) != 2 || repository.messages[1].Content != "测试回答" {
		t.Fatalf("unexpected persisted messages: %+v", repository.messages)
	}
	if len(repository.savedSteps) != 2 || len(events) != 6 || len(repository.savedEvents) != 6 {
		t.Fatalf("steps=%d events=%d", len(repository.savedSteps), len(events))
	}
	if len(runner.input) < 2 || runner.input[0].Role != agent.RoleSystem || runner.input[len(runner.input)-1].Content != "锅炉检查要求" {
		t.Fatalf("unexpected agent input: %+v", runner.input)
	}
}

func TestAskRejectsUnsupportedDataAnalysis(t *testing.T) {
	err := validateAskInput(AskInput{Message: "分析表格", Mode: "data_analysis"})
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeUnsupportedIntent {
		t.Fatalf("error = %v, want unsupported_intent", err)
	}
}

func TestListConversationsNormalizesDocumentedOptions(t *testing.T) {
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", Status: "active"}}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: &fakeAgentRunner{}, prompt: "system"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := qa.ListConversations(context.Background(), "user-id", ConversationListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Page != 1 || result.PageSize != 20 {
		t.Fatalf("page=%d pageSize=%d", result.Page, result.PageSize)
	}
	if _, err = qa.ListConversations(context.Background(), "user-id", ConversationListOptions{Status: "deleted"}); err == nil {
		t.Fatal("expected invalid status to fail")
	}
	if _, err = qa.ListConversations(context.Background(), "user-id", ConversationListOptions{Sort: "title"}); err == nil {
		t.Fatal("expected invalid sort to fail")
	}
}

func TestExtractCitationsFromMessagesAssignsStableNumbers(t *testing.T) {
	score := 0.92
	toolResult := `{"results":[{"documentId":"doc-1","documentName":"手册.pdf","chunkId":"chunk-1","knowledgeBaseId":"kb-1","quoteText":"变压器应保持清洁","score":0.92}]}`
	messages := []agent.Message{
		{Role: agent.RoleSystem, Content: "system"},
		{Role: agent.RoleUser, Content: "question"},
		{Role: agent.RoleAssistant, ToolCalls: []agent.ToolCall{{ID: "call-1", Function: agent.FunctionCall{Name: "search_knowledge"}}}},
		{Role: agent.RoleTool, Name: "search_knowledge", Content: toolResult},
		{Role: agent.RoleAssistant, Content: "answer"},
	}
	citations := extractCitationsFromMessages(messages)
	if len(citations) != 1 {
		t.Fatalf("got %d citations, want 1", len(citations))
	}
	c := citations[0]
	if c.CitationNo != 1 || c.DocumentID != "doc-1" || c.Text != "变压器应保持清洁" || c.Score == nil || *c.Score != score {
		t.Fatalf("unexpected citation: %+v", c)
	}
	if c.Metadata == nil {
		t.Fatal("metadata must not be nil")
	}
}

func TestExtractCitationsFromMessagesHandlesPrefixedToolName(t *testing.T) {
	toolResult := `{"results":[{"documentId":"doc-2","documentName":"规范.pdf","chunkId":"c2","contentPreview":"预览文本"}]}`
	messages := []agent.Message{
		{Role: agent.RoleTool, Name: "knowledge__search_knowledge", Content: toolResult},
	}
	citations := extractCitationsFromMessages(messages)
	if len(citations) != 1 || citations[0].Text != "预览文本" {
		t.Fatalf("unexpected citations from prefixed tool: %+v", citations)
	}
}

func TestExtractCitationsFromMessagesAssignsSequentialNumbers(t *testing.T) {
	result1 := `{"results":[{"documentId":"d1","quoteText":"first"},{"documentId":"d2","quoteText":"second"}]}`
	result2 := `{"results":[{"documentId":"d3","quoteText":"third"}]}`
	messages := []agent.Message{
		{Role: agent.RoleTool, Name: "search_knowledge", Content: result1},
		{Role: agent.RoleTool, Name: "search_knowledge", Content: result2},
	}
	citations := extractCitationsFromMessages(messages)
	if len(citations) != 3 {
		t.Fatalf("got %d citations, want 3", len(citations))
	}
	for i, c := range citations {
		if c.CitationNo != i+1 {
			t.Errorf("citation %d: citationNo=%d, want %d", i, c.CitationNo, i+1)
		}
	}
}

func TestAskSavesCitationsFromSearchKnowledgeToolResult(t *testing.T) {
	now := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepository{conversation: Conversation{ID: "conv-id", OwnerUserID: "u", Title: "新对话", Status: "active", CreatedAt: now, UpdatedAt: now}}
	toolResult := `{"results":[{"documentId":"doc-1","documentName":"手册.pdf","chunkId":"c1","quoteText":"关键片段","knowledgeBaseId":"kb-1"}]}`
	runner := &fakeAgentRunnerWithToolResult{toolResult: toolResult}
	qa, err := NewQAService(repo, fakeRuntimeProvider{runner: runner, prompt: "system"})
	if err != nil {
		t.Fatal(err)
	}
	qa.now = func() time.Time { return now }
	saver := &fakeCitationSaver{}
	qa.SetCitationSaver(saver)
	result, err := qa.Ask(context.Background(), "u", "conv-id", AskInput{Message: "巡检要点"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Citations) != 1 {
		t.Fatalf("expected 1 citation in result, got %d", len(result.Citations))
	}
	if saver.savedMessageID == "" || len(saver.saved) != 1 {
		t.Fatalf("citation not saved: messageID=%q saved=%d", saver.savedMessageID, len(saver.saved))
	}
	if saver.saved[0].CitationNo != 1 || saver.saved[0].DocumentID != "doc-1" {
		t.Fatalf("unexpected saved citation: %+v", saver.saved[0])
	}
}

type fakeCitationSaver struct {
	savedMessageID string
	saved          []Citation
}

func (s *fakeCitationSaver) SaveCitations(_ context.Context, messageID string, citations []Citation) ([]Citation, error) {
	s.savedMessageID = messageID
	s.saved = append([]Citation(nil), citations...)
	return citations, nil
}

// fakeAgentRunnerWithToolResult produces an agent result that includes a
// search_knowledge tool call and result message, simulating knowledge retrieval.
type fakeAgentRunnerWithToolResult struct {
	toolResult string
}

func (r *fakeAgentRunnerWithToolResult) RunWithObserver(_ context.Context, input []agent.Message, _ agent.Observer) (agent.Result, error) {
	toolMsg := agent.Message{Role: agent.RoleTool, Name: "search_knowledge", Content: r.toolResult}
	final := agent.Message{Role: agent.RoleAssistant, Content: "回答内容"}
	return agent.Result{Final: final, Messages: append(append([]agent.Message(nil), input...), toolMsg, final), Iterations: 1}, nil
}

func TestCancelActiveRunCancelsAgentAndPersistsCancelledMessage(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	repository := &fakeRepository{conversation: Conversation{ID: "conversation-id", OwnerUserID: "user-id", Status: "active", CreatedAt: now, UpdatedAt: now}}
	runner := blockingAgentRunner{started: make(chan struct{})}
	qa, err := NewQAService(repository, fakeRuntimeProvider{runner: runner, prompt: "system"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := qa.Ask(context.Background(), "user-id", "conversation-id", AskInput{Message: "cancel me"}, nil)
		done <- err
	}()
	<-runner.started
	qa.CancelActiveRun("run-id")
	if err := <-done; err == nil {
		t.Fatal("expected cancelled ask to fail")
	}
	if got := repository.messages[1].Status; got != "cancelled" {
		t.Fatalf("assistant status=%q", got)
	}
}
