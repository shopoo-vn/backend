package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/marketplace/media-service/internal/imageproc"
	"github.com/marketplace/media-service/internal/model"
	"github.com/marketplace/media-service/internal/repo"
	"github.com/marketplace/media-service/internal/storage"
)

var (
	// ErrNotFound is surfaced when a media row does not exist.
	ErrNotFound = errors.New("media not found")
	// ErrForbidden is returned when a caller acts on media they do not own.
	ErrForbidden = errors.New("not allowed to modify this media")
	// ErrUnsupportedType is returned when the upload is not an image.
	ErrUnsupportedType = errors.New("unsupported content type: image required")
)

// All derived images are normalised to JPEG, so every object key ends in .jpg.
const derivedExt = "jpg"
const derivedContentType = "image/jpeg"

// variants defines the sizes produced for every upload. "full" (Width 0) is the
// re-encoded/compressed original; thumb and medium are downscaled.
var variants = []imageproc.Variant{
	{Name: "thumb", Width: 256},
	{Name: "medium", Width: 1024},
	{Name: "full", Width: 0},
}

// MediaService orchestrates image processing, object storage and metadata.
type MediaService struct {
	repo  *repo.MediaRepo
	store *storage.Store
	proc  *imageproc.Processor
}

func NewMediaService(r *repo.MediaRepo, store *storage.Store, proc *imageproc.Processor) *MediaService {
	return &MediaService{repo: r, store: store, proc: proc}
}

// View is the client-facing media representation with public URLs per size.
type View struct {
	ID        string            `json:"id"`
	OwnerID   string            `json:"owner_id"`
	URLs      map[string]string `json:"urls"`
	CreatedAt string            `json:"created_at"`
}

func objectKey(id, size string) string {
	return fmt.Sprintf("media/%s/%s.%s", id, size, derivedExt)
}

// Upload validates, resizes and stores an image, then persists its metadata.
// data is the raw uploaded bytes; originalName/contentType come from the part.
func (s *MediaService) Upload(ctx context.Context, ownerID string, data []byte, originalName, contentType string) (*model.Media, error) {
	results, err := s.proc.Process(ctx, data, variants)
	if err != nil {
		return nil, fmt.Errorf("process image: %w", err)
	}

	id := uuid.NewString()
	sizes := make(map[string]string, len(results))
	uploaded := make([]string, 0, len(results))

	// Upload every derived size. On any failure, best-effort clean up what we
	// already wrote so we don't leak orphaned objects.
	for _, res := range results {
		key := objectKey(id, res.Name)
		if err := s.store.Put(ctx, key, bytes.NewReader(res.Bytes), int64(len(res.Bytes)), derivedContentType); err != nil {
			s.cleanup(ctx, uploaded)
			return nil, fmt.Errorf("store %s: %w", res.Name, err)
		}
		uploaded = append(uploaded, key)
		sizes[res.Name] = key
	}

	m := &model.Media{
		ID:           id,
		OwnerID:      ownerID,
		OriginalName: originalName,
		ContentType:  contentType,
		Sizes:        sizes,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		s.cleanup(ctx, uploaded)
		return nil, fmt.Errorf("persist media: %w", err)
	}
	return m, nil
}

// Get returns a media record by id.
func (s *MediaService) Get(ctx context.Context, id string) (*model.Media, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

// Delete removes the objects and the metadata row. Only the owner may delete.
func (s *MediaService) Delete(ctx context.Context, id, requesterID string) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if m.OwnerID != requesterID {
		return ErrForbidden
	}

	// Delete objects first; then the row. A failure to remove an object should
	// not block the metadata delete (objects can be garbage-collected later),
	// but a real storage error is still surfaced.
	for _, key := range m.Sizes {
		if err := s.store.Remove(ctx, key); err != nil {
			return fmt.Errorf("delete object: %w", err)
		}
	}
	if _, err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete media row: %w", err)
	}
	return nil
}

// ToView builds the client-facing representation with public URLs.
func (s *MediaService) ToView(m *model.Media) *View {
	urls := make(map[string]string, len(m.Sizes))
	for size, key := range m.Sizes {
		urls[size] = s.store.PublicURL(key)
	}
	return &View{
		ID:        m.ID,
		OwnerID:   m.OwnerID,
		URLs:      urls,
		CreatedAt: m.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	}
}

// cleanup best-effort removes objects already written when a later step fails.
func (s *MediaService) cleanup(ctx context.Context, keys []string) {
	for _, key := range keys {
		_ = s.store.Remove(ctx, key)
	}
}
