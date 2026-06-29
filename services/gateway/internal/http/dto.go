package httpapi

import "time"

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createSessionRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userSummaryResponse struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type sessionSummaryResponse struct {
	SessionID   string `json:"sessionId"`
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresAt   string `json:"expiresAt"`
}

type sessionResponseData struct {
	User    userSummaryResponse    `json:"user"`
	Session sessionSummaryResponse `json:"session"`
}

func dateTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
