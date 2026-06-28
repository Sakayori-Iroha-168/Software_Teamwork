package service

import "strings"

func classifyIntent(message string) (IntentType, float64) {
	normalized := strings.ToLower(message)
	keywords := []string{
		"知识库",
		"规程",
		"规范",
		"文档",
		"引用",
		"检索",
		"rag",
		"标准",
		"条款",
	}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return IntentKnowledgeQA, 0.86
		}
	}
	return IntentGeneralChat, 0.72
}
