package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

type FileRepository struct {
	*MemoryRepository
	path string
}

type fileSnapshot struct {
	Profiles    map[string]service.ModelProfile           `json:"profiles"`
	Credentials map[string]service.ProviderCredential     `json:"credentials"`
	Revisions   map[string][]service.ModelProfileRevision `json:"revisions"`
}

func NewFileRepository(path string) (*FileRepository, error) {
	repo := &FileRepository{
		MemoryRepository: NewMemoryRepository(),
		path:             path,
	}
	if err := repo.load(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *FileRepository) CreateProfile(ctx context.Context, profile service.ModelProfile, credential service.ProviderCredential, revision service.ModelProfileRevision) (service.ModelProfile, error) {
	created, err := r.MemoryRepository.CreateProfile(ctx, profile, credential, revision)
	if err != nil {
		return service.ModelProfile{}, err
	}
	if err := r.persist(); err != nil {
		return service.ModelProfile{}, err
	}
	return created, nil
}

func (r *FileRepository) UpdateProfile(ctx context.Context, profile service.ModelProfile, credential *service.ProviderCredential, revision service.ModelProfileRevision) (service.ModelProfile, error) {
	updated, err := r.MemoryRepository.UpdateProfile(ctx, profile, credential, revision)
	if err != nil {
		return service.ModelProfile{}, err
	}
	if err := r.persist(); err != nil {
		return service.ModelProfile{}, err
	}
	return updated, nil
}

func (r *FileRepository) DeleteProfile(ctx context.Context, id string, deletedAt time.Time, revision service.ModelProfileRevision) error {
	if err := r.MemoryRepository.DeleteProfile(ctx, id, deletedAt, revision); err != nil {
		return err
	}
	return r.persist()
}

func (r *FileRepository) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.path == "" {
		return errors.New("profile store path is empty")
	}
	return nil
}

func (r *FileRepository) load() error {
	if r.path == "" {
		return errors.New("profile store path is empty")
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var snapshot fileSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if snapshot.Profiles != nil {
		r.profiles = snapshot.Profiles
	}
	if snapshot.Credentials != nil {
		r.credentials = snapshot.Credentials
	}
	if snapshot.Revisions != nil {
		r.revisions = snapshot.Revisions
	}
	return nil
}

func (r *FileRepository) persist() error {
	r.mu.RLock()
	snapshot := fileSnapshot{
		Profiles:    make(map[string]service.ModelProfile, len(r.profiles)),
		Credentials: make(map[string]service.ProviderCredential, len(r.credentials)),
		Revisions:   make(map[string][]service.ModelProfileRevision, len(r.revisions)),
	}
	for id, profile := range r.profiles {
		snapshot.Profiles[id] = cloneProfile(profile)
	}
	for id, credential := range r.credentials {
		snapshot.Credentials[id] = cloneCredential(credential)
	}
	for profileID, revisions := range r.revisions {
		snapshot.Revisions[profileID] = append([]service.ModelProfileRevision(nil), revisions...)
	}
	r.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}
