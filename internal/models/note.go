package models

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrInvalid = errors.New("invalid note")

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type NoteCreate struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
type NoteUpdate struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}
