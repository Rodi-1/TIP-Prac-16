package service

import (
	"context"
	"example.com/pz16/internal/models"
	"example.com/pz16/internal/repo"
	"strings"
)

type Service struct{ Notes repo.NoteRepo }

func (s Service) Create(ctx context.Context, n *models.Note) error {
	n.Title = strings.TrimSpace(n.Title)
	if n.Title == "" {
		return models.ErrInvalid
	}
	return s.Notes.Create(ctx, n)
}
func (s Service) Get(ctx context.Context, id int64) (models.Note, error) { return s.Notes.Get(ctx, id) }
func (s Service) List(ctx context.Context, q string, limit, offset int) ([]models.Note, int, error) {
	return s.Notes.List(ctx, q, limit, offset)
}
func (s Service) Patch(ctx context.Context, id int64, in models.NoteUpdate) (models.Note, error) {
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return models.Note{}, models.ErrInvalid
		}
		in.Title = &title
	}
	return s.Notes.Update(ctx, id, in)
}
func (s Service) Delete(ctx context.Context, id int64) error { return s.Notes.Delete(ctx, id) }
