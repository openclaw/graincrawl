package granola

import (
	"encoding/json"
	"os"
	"time"
)

type SupabaseFile struct {
	SessionID    string `json:"session_id"`
	UserInfoRaw  string `json:"user_info"`
	WorkOSTokens string `json:"workos_tokens"`
}

type WorkOSTokens struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	IDToken      string `json:"id_token"`
	ObtainedAt   int64  `json:"obtained_at"`
	RefreshToken string `json:"refresh_token"`
	SessionID    string `json:"session_id"`
	SignInMethod string `json:"sign_in_method"`
	TokenType    string `json:"token_type"`
}

type UserInfo struct {
	ID                string   `json:"id"`
	Email             string   `json:"email"`
	WorkspaceIDs      []string `json:"workspace_ids"`
	ActiveWorkspaceID string   `json:"active_workspace_id"`
}

type TokenSummary struct {
	Present        bool      `json:"present"`
	Expired        bool      `json:"expired"`
	ExpiresAt      time.Time `json:"expires_at,omitempty"`
	RefreshPresent bool      `json:"refresh_present"`
	SignInMethod   string    `json:"sign_in_method,omitempty"`
}

func ReadSupabase(path string) (SupabaseFile, WorkOSTokens, UserInfo, error) {
	var file SupabaseFile
	b, err := os.ReadFile(path)
	if err != nil {
		return file, WorkOSTokens{}, UserInfo{}, err
	}
	if err := json.Unmarshal(b, &file); err != nil {
		return file, WorkOSTokens{}, UserInfo{}, err
	}
	var tokens WorkOSTokens
	if file.WorkOSTokens != "" {
		if err := json.Unmarshal([]byte(file.WorkOSTokens), &tokens); err != nil {
			return file, WorkOSTokens{}, UserInfo{}, err
		}
	}
	var user UserInfo
	if file.UserInfoRaw != "" {
		_ = json.Unmarshal([]byte(file.UserInfoRaw), &user)
	}
	return file, tokens, user, nil
}

func SummarizeToken(tokens WorkOSTokens, now time.Time) TokenSummary {
	expires := time.UnixMilli(tokens.ObtainedAt).Add(time.Duration(tokens.ExpiresIn) * time.Second)
	return TokenSummary{
		Present:        tokens.AccessToken != "",
		Expired:        tokens.AccessToken == "" || now.After(expires),
		ExpiresAt:      expires,
		RefreshPresent: tokens.RefreshToken != "",
		SignInMethod:   tokens.SignInMethod,
	}
}
