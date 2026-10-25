package repo

import (
	"context"
	"database/sql"
	"errors"
	"example.com/pz16/internal/models"
)

type NoteRepo struct{ DB *sql.DB }

func (r NoteRepo) Create(ctx context.Context, n *models.Note) error {
	return r.DB.QueryRowContext(ctx, `INSERT INTO notes(title,content) VALUES($1,$2) RETURNING id,created_at,updated_at`, n.Title, n.Content).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
}
func (r NoteRepo) Get(ctx context.Context, id int64) (models.Note, error) {
	var n models.Note
	e := r.DB.QueryRowContext(ctx, `SELECT id,title,content,created_at,updated_at FROM notes WHERE id=$1`, id).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		e = models.ErrNotFound
	}
	return n, e
}
func (r NoteRepo) List(ctx context.Context, q string, limit, offset int) ([]models.Note, int, error) {
	var total int
	if e := r.DB.QueryRowContext(ctx, `SELECT count(*) FROM notes WHERE title ILIKE '%' || $1 || '%'`, q).Scan(&total); e != nil {
		return nil, 0, e
	}
	rows, e := r.DB.QueryContext(ctx, `SELECT id,title,content,created_at,updated_at FROM notes WHERE title ILIKE '%' || $1 || '%' ORDER BY id LIMIT $2 OFFSET $3`, q, limit, offset)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	out := make([]models.Note, 0)
	for rows.Next() {
		var n models.Note
		if e = rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); e != nil {
			return nil, 0, e
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}
func (r NoteRepo) Update(ctx context.Context, id int64, in models.NoteUpdate) (models.Note, error) {
	var n models.Note
	e := r.DB.QueryRowContext(ctx, `UPDATE notes SET title=COALESCE($2,title),content=COALESCE($3,content) WHERE id=$1 RETURNING id,title,content,created_at,updated_at`, id, in.Title, in.Content).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		e = models.ErrNotFound
	}
	return n, e
}
func (r NoteRepo) Delete(ctx context.Context, id int64) error {
	res, e := r.DB.ExecContext(ctx, `DELETE FROM notes WHERE id=$1`, id)
	if e != nil {
		return e
	}
	count, e := res.RowsAffected()
	if e == nil && count == 0 {
		return models.ErrNotFound
	}
	return e
}
