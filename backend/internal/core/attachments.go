package core

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxUploadBytes int64 = 15 << 20

func (s *Store) SaveAttachment(userID string, header *multipart.FileHeader) (*Attachment, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("login is required")
	}
	if header == nil {
		return nil, errors.New("file is required")
	}
	if header.Size <= 0 {
		return nil, errors.New("file is empty")
	}
	if header.Size > maxUploadBytes {
		return nil, errors.New("file must be 15MB or smaller")
	}

	source, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer source.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.newID("att")
	storedName := id + "-" + slug(header.Filename)
	if filepath.Ext(storedName) == "" {
		storedName += ".bin"
	}
	root, err := filepath.Abs(s.cfg.UploadRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	storedPath := filepath.Join(root, storedName)
	target, err := os.Create(storedPath)
	if err != nil {
		return nil, err
	}
	written, copyErr := io.Copy(target, io.LimitReader(source, maxUploadBytes+1))
	closeErr := target.Close()
	if copyErr != nil {
		return nil, copyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if written > maxUploadBytes {
		_ = os.Remove(storedPath)
		return nil, errors.New("file must be 15MB or smaller")
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(storedPath)
	}
	attachment := &Attachment{
		ID:           id,
		UserID:       userID,
		OriginalName: strings.TrimSpace(header.Filename),
		StoredName:   storedName,
		ContentType:  contentType,
		SizeBytes:    written,
		URL:          "/api/uploads/" + id + "/download",
		StoredPath:   storedPath,
		IsImage:      strings.HasPrefix(strings.ToLower(contentType), "image/"),
		CreatedAt:    time.Now().UTC(),
	}
	s.attachments[id] = attachment
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copyAttachment := *attachment
	return &copyAttachment, nil
}

func (s *Store) AttachmentForDownload(id string) (*Attachment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	attachment, ok := s.attachments[id]
	if !ok {
		return nil, false
	}
	copyAttachment := *attachment
	return &copyAttachment, true
}

func (s *Store) ListAttachments(userID string) []*Attachment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	attachments := make([]*Attachment, 0, len(s.attachments))
	for _, attachment := range s.attachments {
		if userID != "" && attachment.UserID != userID {
			continue
		}
		copyAttachment := *attachment
		attachments = append(attachments, &copyAttachment)
	}
	return attachments
}

func detectContentType(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	return http.DetectContentType(buffer[:n])
}

func cloneAttachment(attachment *Attachment) *Attachment {
	if attachment == nil {
		return nil
	}
	copyAttachment := *attachment
	return &copyAttachment
}
