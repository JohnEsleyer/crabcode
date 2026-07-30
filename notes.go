package main

import (
	"errors"
	"fmt"
	"time"
)

func (a *App) GetNotes() ([]Note, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query("SELECT id, title, content, created_at, updated_at FROM markdown_notes ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func (a *App) CreateNote(title string) (*Note, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf("# %s\n\nWrite your markdown observations here.", title)

	_, err := a.db.Exec(
		"INSERT INTO markdown_notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, title, content, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &Note{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (a *App) SaveNote(id string, title string, content string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec(
		"UPDATE markdown_notes SET title = ?, content = ?, updated_at = ? WHERE id = ?",
		title, content, now, id,
	)
	return err
}

func (a *App) DeleteNote(id string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	_, err := a.db.Exec("DELETE FROM markdown_notes WHERE id = ?", id)
	return err
}
