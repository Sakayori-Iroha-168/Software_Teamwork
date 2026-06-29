package repository_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

func TestFileRepositoryPersistsProfilesAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	repo, err := repository.NewFileRepository(path)
	if err != nil {
		t.Fatalf("NewFileRepository() error = %v", err)
	}
	profile := service.ModelProfile{
		ID:                "mp_chat",
		Name:              "chat",
		Purpose:           service.PurposeChat,
		Provider:          service.ProviderOpenAICompatible,
		BaseURL:           "https://api.example.com/v1",
		Model:             "gpt",
		Enabled:           true,
		IsDefault:         true,
		TimeoutMs:         60000,
		SupportsStreaming: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	credential := service.ProviderCredential{
		ID:                "cred_chat",
		ProfileID:         "mp_chat",
		StorageMode:       "encrypted_column",
		Ciphertext:        []byte("ciphertext"),
		FingerprintSHA256: "fingerprint",
		Status:            "active",
		CreatedAt:         now,
	}
	revision := service.ModelProfileRevision{
		ID:         "rev_chat",
		ProfileID:  "mp_chat",
		ChangeType: "created",
		CreatedAt:  now,
	}
	if _, err := repo.CreateProfile(context.Background(), profile, credential, revision); err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	reloaded, err := repository.NewFileRepository(path)
	if err != nil {
		t.Fatalf("reload NewFileRepository() error = %v", err)
	}
	got, err := reloaded.GetProfile(context.Background(), "mp_chat")
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if got.ID != "mp_chat" || !got.APIKeyConfigured {
		t.Fatalf("profile = %+v", got)
	}
}
