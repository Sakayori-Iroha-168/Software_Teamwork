/**
 * Shared TypeScript types aligned with the Gateway OpenAPI specification.
 *
 * All field names use camelCase matching the spec schemas exactly.
 * Type names match the OpenAPI schema names where applicable.
 */

// =============================================================================
// Common
// =============================================================================

export interface PageInfo {
  page: number
  pageSize: number
  total: number
}

/** @deprecated Use PageInfo instead. */
export type ListResult<T> = {
  items: T[]
  total: number
  page: number
  pageSize: number
}

// =============================================================================
// Auth
// =============================================================================

export interface UserSummary {
  id: string
  username: string
  roles: string[]
  permissions: string[]
}

export interface SessionSummary {
  sessionId: string
  accessToken: string
  tokenType: 'Bearer'
  expiresAt: string
}

export interface CreateUserRequest {
  username: string
  password: string
}

export interface CreateSessionRequest {
  username: string
  password: string
}

// =============================================================================
// QA Sessions
// =============================================================================

export type QASessionStatus = 'active' | 'archived'

export interface QASession {
  sessionId: string
  title?: string
  status: QASessionStatus
  messageCount?: number
  lastMessagePreview?: string
  createdAt: string
  updatedAt: string
}

export interface QASessionListItem {
  sessionId: string
  title?: string
  status: QASessionStatus
  messageCount?: number
  lastMessagePreview?: string
  createdAt: string
  updatedAt: string
}

export interface CreateQASessionRequest {
  title?: string
}

export interface UpdateQASessionRequest {
  title?: string
  status?: QASessionStatus
}

// =============================================================================
// QA Messages
// =============================================================================

export type QAMessageRole = 'user' | 'assistant' | 'system'

export type QAMessageStatus =
  'pending' | 'queued' | 'streaming' | 'completed' | 'stopped' | 'failed' | 'cancelled'

export type QAIntent =
  'knowledge_qa' | 'general_chat' | 'report_generation' | 'data_analysis' | 'unknown'

export interface QAMessage {
  messageId: string
  sessionId: string
  sequenceNo?: number
  role: QAMessageRole
  status: QAMessageStatus
  intent?: QAIntent
  content: string
  thinking?: QAThinkingStep[]
  citations?: QACitation[]
  createdAt: string
  completedAt?: string | null
}

export interface QARetrievalOptions {
  topK?: number
  scoreThreshold?: number
  /** @deprecated Backward-compatible alias for scoreThreshold. */
  similarityThreshold?: number
  enableRerank?: boolean
  /** @deprecated Backward-compatible alias for enableRerank. */
  useRerank?: boolean
  rerankThreshold?: number
  rerankTopN?: number
  tagFilters?: Record<string, string[]>
}

export interface QAAgentOptions {
  enabledToolNames?: string[]
  maxIterations?: number
}

export interface CreateQAMessageRequest {
  /** Answer-question text. */
  message: string
  /** Forced intent mode. */
  mode?: QAIntent
  /** Knowledge base IDs to retrieve from. */
  knowledgeBaseIds?: string[]
  /** Retrieval parameter overrides. */
  retrieval?: QARetrievalOptions
  /** Agent runtime overrides. */
  agent?: QAAgentOptions
}

// =============================================================================
// QA Thinking Steps
// =============================================================================

export type QAThinkingStepType =
  'agent_iteration' | 'tool_call' | 'tool_result' | 'generation' | 'citation' | 'verify'

export type QAThinkingStepStatus = 'pending' | 'running' | 'done' | 'failed'

export interface QAThinkingStep {
  type: QAThinkingStepType
  label?: string
  status: QAThinkingStepStatus
  /** Sanitized user-visible detail. */
  detail?: string
}

// =============================================================================
// QA Citations
// =============================================================================

export interface QACitation {
  citationId: string
  messageId: string
  citationNo?: number
  documentId?: string
  /** @deprecated Backward-compatible alias for documentId. */
  docId?: string
  documentName?: string
  /** @deprecated Backward-compatible alias for documentName. */
  docName?: string
  knowledgeBaseId?: string
  chunkId?: string
  sectionPath?: string
  /** Citation quote preview / saved quote text. */
  text?: string
  contentPreview?: string
  context?: string
  pageNumber?: number
  score?: number
  rerankScore?: number | null
  chunkType?: string
  isSourceAvailable?: boolean
  metadata?: Record<string, unknown>
}

export interface QACitationDetail extends QACitation {
  content?: string
  source?: {
    available: boolean
    reason?: string
    downloadEndpoint?: string
  }
}

export interface CreateQACitationLookupRequest {
  citationIds: string[]
}

// =============================================================================
// QA Response Runs
// =============================================================================

export type QAResponseRunStatus =
  'queued' | 'running' | 'streaming' | 'completed' | 'failed' | 'cancelled'

export type QATerminationReason =
  | 'completed'
  | 'max_iterations'
  | 'timeout'
  | 'cancelled'
  | 'tool_error'
  | 'model_error'
  | 'policy_denied'

export interface QAResponseRun {
  id: string
  sessionId: string
  userMessageId?: string
  assistantMessageId?: string
  status: QAResponseRunStatus
  currentIteration?: number
  maxIterations?: number
  terminationReason?: QATerminationReason | null
  totalTokens?: number
  latencyMs?: number
  createdAt: string
  completedAt?: string | null
}

export interface UpdateQAResponseRunRequest {
  status: 'cancelled'
}

// =============================================================================
// QA Agent Tool Calls
// =============================================================================

export type QAAgentToolCallStatus = 'running' | 'completed' | 'failed' | 'cancelled'

export interface QAAgentToolCall {
  id: string
  responseRunId: string
  modelInvocationId?: string
  iterationNo: number
  toolCallId: string
  toolName: string
  argumentsSummary?: Record<string, unknown>
  resultSummary?: Record<string, unknown>
  status: QAAgentToolCallStatus
  latencyMs?: number
  startedAt?: string
  finishedAt?: string | null
}

// =============================================================================
// SSE Events
// =============================================================================

export type QAMessageEventType =
  | 'message.created'
  | 'agent.iteration.started'
  | 'reasoning.step'
  | 'tool.started'
  | 'tool.completed'
  | 'tool.failed'
  | 'answer.delta'
  | 'citation.delta'
  | 'answer.completed'
  | 'error'
  | 'heartbeat'

export interface QAMessageEvent {
  eventSeq: number
  eventType: QAMessageEventType
  payload: Record<string, unknown>
  createdAt: string
}

// ── SSE data payload types ──

export interface MessageCreatedData {
  messageId: string
  sessionId: string
  responseRunId: string
  sequenceNo: number
}

export interface AgentIterationStartedData {
  responseRunId: string
  iterationNo: number
}

export interface ReasoningStepData {
  responseRunId: string
  step: QAThinkingStep
}

export interface ToolStartedData {
  responseRunId: string
  toolCallId: string
  toolName: string
}

export interface ToolCompletedData {
  responseRunId: string
  toolCallId: string
  toolName: string
  latencyMs: number
}

export interface ToolFailedData {
  responseRunId: string
  toolCallId: string
  toolName: string
  errorCode: string
  errorMessage: string
}

export interface AnswerDeltaData {
  responseRunId: string
  content: string
  sequenceNo: number
}

export interface CitationDeltaData {
  responseRunId: string
  citation: QACitation
}

export interface AnswerCompletedData {
  responseRunId: string
  assistantMessageId: string
  totalTokens: number
  latencyMs: number
}

export interface SSEErrorData {
  responseRunId?: string
  code: string
  message: string
  fatal: boolean
}

// =============================================================================
// Knowledge Bases
// =============================================================================

export interface ChunkStrategy {
  type?: string
  chunkSize?: number
  overlap?: number
  separators?: string[]
  [key: string]: unknown
}

export interface RetrievalStrategy {
  mode?: string
  topK?: number
  scoreThreshold?: number
  rerankTopN?: number | null
  [key: string]: unknown
}

export interface KnowledgeBaseSummary {
  id: string
  name: string
  description: string
  docType: string
  chunkStrategy: ChunkStrategy
  retrievalStrategy: RetrievalStrategy
  documentCount: number
  chunkCount: number
  createdBy?: string
  createdAt: string
  updatedAt: string
}

export interface CreateKnowledgeBaseRequest {
  /** Optional client-supplied ID. */
  id?: string
  name: string
  description?: string
  docType?: string
  chunkStrategy?: ChunkStrategy
  retrievalStrategy?: RetrievalStrategy
}

export interface UpdateKnowledgeBaseRequest {
  name?: string
  description?: string
  docType?: string
  chunkStrategy?: ChunkStrategy
  retrievalStrategy?: RetrievalStrategy
}

// =============================================================================
// Documents
// =============================================================================

export type DocumentStatus = 'uploaded' | 'parsing' | 'chunking' | 'embedding' | 'ready' | 'failed'

export interface DocumentSummary {
  id: string
  knowledgeBaseId: string
  name: string
  contentType?: string | null
  sizeBytes?: number
  status: DocumentStatus
  errorCode?: string | null
  errorMessage?: string | null
  chunkCount: number
  tags?: string[]
  parserBackend?: string | null
  createdBy?: string | null
  createdAt: string
  updatedAt?: string | null
  jobId?: string | null
}

export interface UpdateDocumentRequest {
  tags?: string[]
}

export interface DocumentChunk {
  id: string
  knowledgeBaseId: string
  documentId: string
  chunkIndex: number
  sectionPath?: string | null
  content: string
  tokenCount: number
  chunkType?: string | null
  qdrantPointId?: string | null
  embeddingProvider?: string | null
  embeddingDimension?: number | null
  embeddingPreview?: number[] | null
  metadata?: Record<string, unknown>
  createdAt: string
}

// =============================================================================
// Knowledge Queries
// =============================================================================

export interface KnowledgeQueryRequest {
  query: string
  knowledgeBaseIds?: string[]
  topK?: number
  scoreThreshold?: number
  tags?: string[]
  metadataFilter?: Record<string, string>
  rerank?: boolean
  rerankTopN?: number | null
}

export interface KnowledgeQueryResult {
  score: number
  pointId?: string
  knowledgeBaseId: string
  documentId: string
  chunkId: string
  documentName: string
  sectionPath?: string | null
  chunkIndex?: number | null
  contentPreview: string
  tags?: string[]
}

export interface KnowledgeQueryTrace {
  embeddingProvider: string
  embeddingModel: string
  embeddingDimension: number
  qdrantCollection: string
  searchTopK: number
  scoreThreshold: number
  hitCount: number
  rerank: boolean
  rerankTopN?: number | null
}

export interface KnowledgeQuerySummary {
  id: string
  query: string
  results: KnowledgeQueryResult[]
  trace: KnowledgeQueryTrace
}

// =============================================================================
// Admin: Model Profiles
// =============================================================================

export type ModelPurpose = 'chat' | 'embedding' | 'rerank'

export type ModelProvider = 'openai_compatible' | 'siliconflow' | 'local_compatible'

export interface ModelProfile {
  id: string
  name: string
  purpose: ModelPurpose
  provider: ModelProvider
  baseUrl: string
  model: string
  enabled: boolean
  isDefault: boolean
  timeoutMs: number
  apiKeyConfigured: boolean
  supportsStreaming: boolean
  dimensions?: number | null
  topN?: number | null
  defaultParameters?: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface CreateModelProfileRequest {
  name: string
  purpose: ModelPurpose
  provider: ModelProvider
  baseUrl: string
  model: string
  /** Write-only provider credential. */
  apiKey: string
  enabled?: boolean
  isDefault?: boolean
  timeoutMs?: number
  supportsStreaming?: boolean
  dimensions?: number | null
  topN?: number | null
  defaultParameters?: Record<string, unknown>
}

export interface UpdateModelProfileRequest {
  name?: string
  provider?: ModelProvider
  baseUrl?: string
  model?: string
  apiKey?: string
  enabled?: boolean
  isDefault?: boolean
  timeoutMs?: number
  supportsStreaming?: boolean
  dimensions?: number | null
  topN?: number | null
  defaultParameters?: Record<string, unknown>
}

// =============================================================================
// Admin: Parser Configs
// =============================================================================

export type ParserBackend = 'builtin' | 'tika' | 'unstructured' | 'local_ocr' | 'remote_compatible'

export interface ParserConfig {
  id: string
  name: string
  backend: ParserBackend
  enabled: boolean
  isDefault: boolean
  concurrency: number
  supportedContentTypes?: string[]
  endpointUrl?: string | null
  defaultParameters?: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface CreateParserConfigRequest {
  name: string
  backend: ParserBackend
  concurrency: number
  enabled?: boolean
  isDefault?: boolean
  supportedContentTypes?: string[]
  endpointUrl?: string | null
  defaultParameters?: Record<string, unknown>
}

export interface UpdateParserConfigRequest {
  name?: string
  backend?: ParserBackend
  enabled?: boolean
  isDefault?: boolean
  concurrency?: number
  supportedContentTypes?: string[]
  endpointUrl?: string | null
  defaultParameters?: Record<string, unknown>
}

// =============================================================================
// Report Types (abridged — key models only)
// =============================================================================

export type ReportStatus =
  | 'draft'
  | 'outline_generating'
  | 'outline_generated'
  | 'content_generating'
  | 'generated'
  | 'exporting'
  | 'exported'
  | 'failed'
  | 'deleted'

export type ReportJobStatus =
  'pending' | 'running' | 'succeeded' | 'partial_succeeded' | 'failed' | 'canceled'

export type ReportJobType =
  | 'outline_generation'
  | 'outline_regeneration'
  | 'content_generation'
  | 'content_regeneration'
  | 'section_regeneration'
  | 'report_file_creation'

export interface ReportType {
  code: string
  name: string
  description?: string
  enabled: boolean
  defaultTemplateId?: string
}

export interface ReportTemplate {
  id: string
  templateName: string
  reportType: string
  version: number
  description?: string
  enabled: boolean
  filename?: string
  fileSize?: number
  createdBy?: string
  createdAt: string
  updatedAt?: string
}

export interface Report {
  id: string
  name: string
  reportType: string
  templateId?: string
  topic?: string
  specialty?: string
  businessObject?: string
  year?: number
  status: ReportStatus
  extraContext?: Record<string, unknown>
  creatorId?: string
  creatorName?: string
  source?: string
  latestJobId?: string
  latestReportFileId?: string
  generatedAt?: string
  exportedAt?: string
  createdAt: string
  updatedAt?: string
}

export interface ReportJob {
  id: string
  reportId: string
  templateId?: string
  jobType: ReportJobType
  targetType?: string
  targetId?: string
  status: ReportJobStatus
  progress?: Record<string, unknown>
  resultSummary?: string
  error?: ReportJobError
  startedAt?: string
  finishedAt?: string
  createdAt: string
}

export interface ReportJobError {
  code?: string
  message?: string
}

export interface ReportFile {
  id: string
  reportId: string
  jobId?: string
  filename?: string
  format: 'docx'
  fileSize?: number
  status: ReportJobStatus
  contentPath?: string
  createdBy?: string
  createdAt: string
}

// =============================================================================
// QA Config / LLM Config
// =============================================================================

export interface QALLMConfigVersion {
  id: string
  versionNo: number
  /** Always "ai-gateway" in the current contract. */
  provider: 'ai-gateway'
  /** AI Gateway chat model profile id. */
  profileId: string
  modelName: string
  timeoutSeconds?: number
  temperature?: number
  maxTokens?: number
  isActive: boolean
  createdAt: string
}

export interface CreateQALLMConfigVersionRequest {
  provider: 'ai-gateway'
  profileId: string
  modelName: string
  timeoutSeconds?: number
  temperature?: number
  maxTokens?: number
  activate?: boolean
}

export interface QAAgentConfig {
  maxIterations?: number
  toolTimeoutSeconds?: number
  modelTimeoutSeconds?: number
  overallTimeoutSeconds?: number
  enabledToolNames?: string[]
}

export interface QAConfigKnowledgeBase {
  id: string
  type?: string
  displayName?: string
  sortOrder?: number
}

export interface QAConfigVersion {
  id: string
  versionNo: number
  defaultKnowledgeBaseIds?: string[]
  knowledgeBases?: QAConfigKnowledgeBase[]
  retrieval?: QARetrievalOptions
  maxIterations?: number
  toolTimeoutSeconds?: number
  modelTimeoutSeconds?: number
  overallTimeoutSeconds?: number
  enabledToolNames?: string[]
  llm?: QALLMConfigVersion
  agent?: QAAgentConfig
  isActive: boolean
  createdAt: string
}

export interface CreateQAConfigVersionRequest {
  defaultKnowledgeBaseIds?: string[]
  knowledgeBases?: QAConfigKnowledgeBase[]
  retrieval?: QARetrievalOptions
  maxIterations?: number
  toolTimeoutSeconds?: number
  modelTimeoutSeconds?: number
  overallTimeoutSeconds?: number
  enabledToolNames?: string[]
  llm?: CreateQALLMConfigVersionRequest
  agent?: QAAgentConfig
  activate?: boolean
}

export interface QALLMConnectionTestRequest {
  provider: 'ai-gateway'
  profileId: string
  modelName: string
  timeoutSeconds?: number
}

export interface QALLMConnectionTest {
  id: string
  success: boolean
  latencyMs?: number
  modelName?: string
  errorCode?: string
  errorMessage?: string
  testedAt: string
}

// =============================================================================
// QA Retrieval Test
// =============================================================================

export interface QARetrievalTestRunRequest {
  question: string
  /** @deprecated Backward-compatible alias for question. */
  query?: string
  knowledgeBaseIds?: string[]
  retrieval?: QARetrievalOptions
  /** @deprecated Backward-compatible alias for retrieval. */
  overrides?: QARetrievalOptions
}

export interface QARetrievalTestResult {
  rankNo: number
  knowledgeBaseId?: string
  documentId?: string
  /** @deprecated Backward-compatible alias for documentId. */
  docId?: string
  documentName?: string
  /** @deprecated Backward-compatible alias for documentName. */
  docName?: string
  chunkId?: string
  sectionPath?: string
  score?: number
  vectorScore?: number
  rerankScore?: number
  contentPreview?: string
  /** @deprecated Backward-compatible alias for contentPreview. */
  text?: string
  metadata?: Record<string, unknown>
}

export interface QARetrievalTestRun {
  id: string
  question?: string
  /** @deprecated Backward-compatible alias for question. */
  query?: string
  status: 'running' | 'completed' | 'failed'
  results?: QARetrievalTestResult[]
  createdAt: string
  finishedAt?: string | null
}

// =============================================================================
// QA Metrics
// =============================================================================

export interface QAMetricsOverview {
  totalQaCount?: number
  todayQaCount?: number
  totalQuestionCount?: number
  conversationCount?: number
  avgLatencyMs?: number
  activeUsersToday?: number
  knowledgeBaseCount?: number
  documentCount?: number
}

export interface QAMetricsTrendPoint {
  date: string
  count?: number
  questionCount?: number
}

export interface QAMetricsTrend {
  days: number
  points: QAMetricsTrendPoint[]
  /** @deprecated */
  trend30d?: QAMetricsTrendPoint[]
}

export interface QATopQuery {
  query: string
  count: number
  avgLatencyMs?: number
  lastAskedAt?: string
}

export interface QAIntentDistributionItem {
  intent: QAIntent
  label?: string
  count: number
  percent?: number
}

// =============================================================================
// Admin UI (non-API type)
// =============================================================================

export interface AdminMenuItem {
  key: string
  label: string
  icon?: string
  path?: string
  children?: AdminMenuItem[]
}

// =============================================================================
// Backward-compatibility aliases
//
// Map old frontend type names to new OpenAPI-aligned names.
// Consumers can migrate incrementally.
// =============================================================================

/** @deprecated Use `QASession` instead. */
export type Conversation = QASession

/** @deprecated Use `QASessionListItem` instead. */
export type ConversationListItem = QASessionListItem

/** @deprecated Use `QAMessage` instead. */
export type Message = QAMessage

/** @deprecated Use `QAThinkingStep` instead. */
export type ThinkingStep = QAThinkingStep

/** @deprecated Use `QACitation` instead. */
export type Citation = QACitation

/** @deprecated Use `CreateQAMessageRequest` instead. */
export type ChatStreamRequest = CreateQAMessageRequest

/** @deprecated Use `QAIntent` instead. */
export type IntentType = QAIntent

/** @deprecated Use `QAMetricsOverview` instead. */
export type StatsOverview = QAMetricsOverview

/** @deprecated Use `QAMetricsTrendPoint` instead. */
export type TrendPoint = QAMetricsTrendPoint

/** @deprecated Use `QATopQuery` instead. */
export type TopQuery = QATopQuery

// ── Deprecated SSE types (backward compat) ──

/** @deprecated Use QAMessageEventType instead. */
export type SSEEventType = QAMessageEventType

/** @deprecated Use ReasoningStepData + ThinkingStep types instead. */
export interface SSEIntentStatusData {
  status: 'started' | 'done'
  label: string
  intent?: QAIntent
  confidence?: number
  seq: number
}

/** @deprecated Use ReasoningStepData instead. */
export interface SSEThinkingStepData {
  step: QAThinkingStep
  seq: number
}

/** @deprecated Use AnswerDeltaData instead. */
export interface SSETokenData {
  text: string
  index: number
  seq: number
}

/** @deprecated Use CitationDeltaData instead. */
export interface SSECitationData {
  citation: QACitation
  seq: number
}

/** @deprecated Use AnswerCompletedData instead. */
export interface SSEDoneData {
  messageId: string
  assistantMessageId?: string
  totalTokens: number
  promptTokens?: number
  completionTokens?: number
  latencyMs: number
  seq: number
}

// ── Deprecated RAG types (backward compat) ──

/** @deprecated Use KnowledgeQueryRequest instead. */
export interface RAGSearchRequest {
  query: string
  knowledgeBaseIds?: string[]
  topK?: number
  similarityThreshold?: number
  useRerank?: boolean
  rerankThreshold?: number
  rerankTopN?: number
}

/** @deprecated Use KnowledgeQueryResult instead. */
export interface RAGSearchResult {
  rank: number
  chunkId: string
  documentId: string
  documentName: string
  contentPreview: string
  vectorScore: number
  rerankScore?: number
  pageNumber?: number
  chunkIndex?: number
}

// ── Deprecated config types (backward compat) ──

/** @deprecated Use KnowledgeBaseSummary instead. */
export interface KnowledgeBaseConfig {
  id: string
  name: string
  docCount: number
  status: 'active' | 'inactive'
}

/** @deprecated Use QARetrievalOptions instead. */
export interface RAGDefaults {
  knowledgeBases: string[]
  topK: number
  similarityThreshold: number
  useRerank: boolean
  rerankThreshold: number
  rerankTopN: number
}

/**
 * @deprecated Use QALLMConfigVersion instead.
 * Note: the OpenAPI spec no longer exposes apiUrl/apiKey on public
 * LLM config — those are owned by AI Gateway model profiles.
 */
export interface LLMConfig {
  apiUrl: string
  apiKey: string
  modelName: string
  timeout: number
  temperature: number
  maxTokens: number
}

// ── Deprecated document types (backward compat) ──

/** @deprecated Use DocumentSummary instead. */
export interface DocumentUploadResponse {
  docId: string
  filename: string
  fileSize: number
  status: DocumentStatus
  knowledgeBase: string
  uploadedAt: string
}

/** @deprecated Use DocumentSummary.status instead. */
export interface DocumentStatusResponse {
  docId: string
  status: DocumentStatus
  progress: {
    step: string
    current: number
    total: number
    percent: number
  }
  error: string | null
}

/** @deprecated Use QAIntent type directly. */
export interface IntentResult {
  intent: QAIntent
  label: string
  confidence: number
  reasoning?: string
}
