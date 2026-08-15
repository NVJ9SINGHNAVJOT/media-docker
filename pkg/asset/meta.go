package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Meta is the metadata written alongside every asset.
//
// It exists because the raw upload keeps its original extension, so nothing
// else on disk records what that extension is. It also carries the Dispatched
// flag, which distinguishes an upload that has been handed off for processing
// from one the caller abandoned midway (see the janitor in the server).
type Meta struct {
	ID   string `json:"id"`   // Asset id; also the directory name
	Type string `json:"type"` // Storage category (video, image, audio, document, other)
	Ext  string `json:"ext"`  // Extension of original.<ext>, without the dot

	// OriginalName is the file name supplied by the client. It is used only for
	// the download file name of document and other assets, never to build a path.
	OriginalName string `json:"originalName,omitempty"`

	// Dispatched records that the caller completed the upload by calling a
	// dispatch endpoint. Assets left undispatched are reaped by the janitor.
	Dispatched bool `json:"dispatched"`

	// Job records which processing job was dispatched, for diagnostics.
	// Empty for document and other, which are never processed.
	Job string `json:"job,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// WriteMeta writes an asset's metadata atomically.
//
// The write goes to a temporary file that is then renamed into place, so a
// reader never observes a partially written meta.json.
func WriteMeta(m Meta) error {
	encoded, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error encoding meta for %s/%s: %w", m.Type, m.ID, err)
	}

	final := MetaPath(m.Type, m.ID)
	tmp := final + ".tmp"

	if err = os.WriteFile(tmp, encoded, 0644); err != nil {
		return fmt.Errorf("error writing meta for %s/%s: %w", m.Type, m.ID, err)
	}

	if err = os.Rename(tmp, final); err != nil {
		// Best effort cleanup; the janitor will reap the directory if the asset
		// never becomes usable.
		os.Remove(tmp)
		return fmt.Errorf("error committing meta for %s/%s: %w", m.Type, m.ID, err)
	}

	return nil
}

// ReadMeta loads an asset's metadata.
// It returns ErrNotFound if the asset directory or its meta.json does not exist.
func ReadMeta(mediaType, id string) (Meta, error) {
	var m Meta

	content, err := os.ReadFile(MetaPath(mediaType, id))
	if err != nil {
		if os.IsNotExist(err) {
			return m, ErrNotFound
		}
		return m, fmt.Errorf("error reading meta for %s/%s: %w", mediaType, id, err)
	}

	if err = json.Unmarshal(content, &m); err != nil {
		return m, fmt.Errorf("error decoding meta for %s/%s: %w", mediaType, id, err)
	}

	return m, nil
}

// MarkDispatched flags an asset as handed off for processing, so the janitor
// stops treating it as an abandoned upload. The job name is recorded for
// diagnostics.
func MarkDispatched(mediaType, id, job string) (Meta, error) {
	m, err := ReadMeta(mediaType, id)
	if err != nil {
		return m, err
	}

	m.Dispatched = true
	m.Job = job

	if err = WriteMeta(m); err != nil {
		return m, err
	}

	return m, nil
}
