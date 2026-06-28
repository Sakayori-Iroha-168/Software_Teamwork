package httpx

import (
	"net/http"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type Server struct {
	chat          *ChatHandler
	conversations *ConversationHandler
}

func NewServer(
	chat *service.ChatStreamService,
	conversations *service.ConversationService,
) *Server {
	return &Server{
		chat:          NewChatHandler(chat),
		conversations: NewConversationHandler(conversations),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/chat/stream", s.chat.Stream)
	mux.HandleFunc("/api/conversations", s.routeConversations)
	mux.HandleFunc("/api/conversations/", s.routeConversationByID)
	return mux
}

func (s *Server) routeConversations(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/conversations" {
		WriteError(w, http.StatusNotFound, 40400, "not found")
		return
	}
	switch r.Method {
	case http.MethodPost:
		s.conversations.Create(w, r)
	default:
		WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
	}
}

func (s *Server) routeConversationByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
	if path == "" || strings.Contains(path, "/") {
		WriteError(w, http.StatusNotFound, 40400, "not found")
		return
	}
	if r.Method == http.MethodGet {
		s.conversations.Get(w, r)
		return
	}
	WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
}
