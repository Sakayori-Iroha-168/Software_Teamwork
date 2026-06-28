package service

import "testing"

func TestClassifyIntent(t *testing.T) {
	tests := []struct {
		name string
		text string
		want IntentType
	}{
		{name: "knowledge question", text: "帮我检索知识库里的规程条款", want: IntentKnowledgeQA},
		{name: "general chat", text: "你好，介绍一下你自己", want: IntentGeneralChat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := classifyIntent(tt.text)
			if got != tt.want {
				t.Fatalf("classifyIntent() = %s, want %s", got, tt.want)
			}
		})
	}
}
