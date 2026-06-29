package repository

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	profiles    map[string]service.ModelProfile
	credentials map[string]service.ProviderCredential
	revisions   map[string][]service.ModelProfileRevision
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		profiles:    map[string]service.ModelProfile{},
		credentials: map[string]service.ProviderCredential{},
		revisions:   map[string][]service.ModelProfileRevision{},
	}
}

func (r *MemoryRepository) CreateProfile(ctx context.Context, profile service.ModelProfile, credential service.ProviderCredential, revision service.ModelProfileRevision) (service.ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return service.ModelProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.profiles[profile.ID]; exists {
		return service.ModelProfile{}, service.ErrConflict
	}
	if r.nameExistsLocked(profile.Purpose, profile.Name, "") {
		return service.ModelProfile{}, service.ErrConflict
	}
	if profile.Enabled && profile.IsDefault {
		r.clearDefaultLocked(profile.Purpose, profile.ID, profile.UpdatedAt)
	}
	credential.ProfileID = profile.ID
	r.credentials[credential.ID] = cloneCredential(credential)
	profile.CredentialID = &credential.ID
	profile.APIKeyConfigured = true
	r.profiles[profile.ID] = cloneProfile(profile)
	r.appendRevisionLocked(profile.ID, revision)
	return cloneProfile(profile), nil
}

func (r *MemoryRepository) ListProfiles(ctx context.Context, filter service.ListFilter) ([]service.ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]service.ModelProfile, 0, len(r.profiles))
	for _, profile := range r.profiles {
		if profile.DeletedAt != nil {
			continue
		}
		if filter.Purpose != nil && profile.Purpose != *filter.Purpose {
			continue
		}
		if filter.Enabled != nil && profile.Enabled != *filter.Enabled {
			continue
		}
		items = append(items, cloneProfile(profile))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func (r *MemoryRepository) GetProfile(ctx context.Context, id string) (service.ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return service.ModelProfile{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile, exists := r.profiles[id]
	if !exists || profile.DeletedAt != nil {
		return service.ModelProfile{}, service.ErrNotFound
	}
	return cloneProfile(profile), nil
}

func (r *MemoryRepository) UpdateProfile(ctx context.Context, profile service.ModelProfile, credential *service.ProviderCredential, revision service.ModelProfileRevision) (service.ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return service.ModelProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, exists := r.profiles[profile.ID]
	if !exists || current.DeletedAt != nil {
		return service.ModelProfile{}, service.ErrNotFound
	}
	if r.nameExistsLocked(profile.Purpose, profile.Name, profile.ID) {
		return service.ModelProfile{}, service.ErrConflict
	}
	if profile.Enabled && profile.IsDefault {
		r.clearDefaultLocked(profile.Purpose, profile.ID, profile.UpdatedAt)
	}
	if credential != nil {
		now := profile.UpdatedAt
		for id, stored := range r.credentials {
			if stored.ProfileID == profile.ID && stored.Status == "active" {
				stored.Status = "rotated"
				stored.RotatedAt = &now
				r.credentials[id] = stored
			}
		}
		credential.ProfileID = profile.ID
		r.credentials[credential.ID] = cloneCredential(*credential)
		profile.CredentialID = &credential.ID
		profile.APIKeyConfigured = true
	}
	if profile.CredentialID == nil {
		profile.CredentialID = current.CredentialID
	}
	if !profile.APIKeyConfigured {
		profile.APIKeyConfigured = current.APIKeyConfigured
	}
	r.profiles[profile.ID] = cloneProfile(profile)
	r.appendRevisionLocked(profile.ID, revision)
	return cloneProfile(profile), nil
}

func (r *MemoryRepository) DeleteProfile(ctx context.Context, id string, deletedAt time.Time, revision service.ModelProfileRevision) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, exists := r.profiles[id]
	if !exists || profile.DeletedAt != nil {
		return service.ErrNotFound
	}
	profile.Enabled = false
	profile.IsDefault = false
	profile.DeletedAt = &deletedAt
	profile.UpdatedAt = deletedAt
	r.profiles[id] = cloneProfile(profile)
	r.appendRevisionLocked(id, revision)
	return nil
}

func (r *MemoryRepository) Ping(ctx context.Context) error {
	return ctx.Err()
}

func (r *MemoryRepository) nameExistsLocked(purpose service.ModelPurpose, name string, excludingID string) bool {
	for _, profile := range r.profiles {
		if profile.ID != excludingID && profile.DeletedAt == nil && profile.Purpose == purpose && profile.Name == name {
			return true
		}
	}
	return false
}

func (r *MemoryRepository) clearDefaultLocked(purpose service.ModelPurpose, excludingID string, updatedAt time.Time) {
	for id, profile := range r.profiles {
		if id != excludingID && profile.DeletedAt == nil && profile.Purpose == purpose && profile.Enabled && profile.IsDefault {
			profile.IsDefault = false
			profile.UpdatedAt = updatedAt
			r.profiles[id] = profile
		}
	}
}

func (r *MemoryRepository) appendRevisionLocked(profileID string, revision service.ModelProfileRevision) {
	revision.ProfileID = profileID
	revision.RevisionNo = len(r.revisions[profileID]) + 1
	r.revisions[profileID] = append(r.revisions[profileID], revision)
}

func cloneProfile(profile service.ModelProfile) service.ModelProfile {
	if profile.Dimensions != nil {
		value := *profile.Dimensions
		profile.Dimensions = &value
	}
	if profile.TopN != nil {
		value := *profile.TopN
		profile.TopN = &value
	}
	if profile.CredentialID != nil {
		value := *profile.CredentialID
		profile.CredentialID = &value
	}
	if profile.DeletedAt != nil {
		value := *profile.DeletedAt
		profile.DeletedAt = &value
	}
	if profile.DefaultParameters != nil {
		profile.DefaultParameters = append(profile.DefaultParameters[:0:0], profile.DefaultParameters...)
	}
	return profile
}

func cloneCredential(credential service.ProviderCredential) service.ProviderCredential {
	if credential.Ciphertext != nil {
		credential.Ciphertext = append(credential.Ciphertext[:0:0], credential.Ciphertext...)
	}
	if credential.RotatedAt != nil {
		value := *credential.RotatedAt
		credential.RotatedAt = &value
	}
	return credential
}
