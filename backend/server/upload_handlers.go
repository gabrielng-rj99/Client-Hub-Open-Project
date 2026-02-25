package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Open-Generic-Hub/backend/store"
)

// Allowed MIME types for image uploads
var allowedMimeTypes = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
}

// Magic bytes for file type validation
var imageMagicBytes = map[string][]byte{
	"image/jpeg":    {0xFF, 0xD8, 0xFF},
	"image/png":     {0x89, 0x50, 0x4E, 0x47},
	"image/gif":     {0x47, 0x49, 0x46},
	"image/webp":    {0x52, 0x49, 0x46, 0x46}, // RIFF header
	"image/svg+xml": {0x3C, 0x73, 0x76, 0x67}, // <svg
}

// HandleUpload handles file uploads
// POST /api/upload
// Requires Auth (Root only)
func (s *Server) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// check method
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user info for audit logging
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// 15MB limit (matches frontend)
	maxSize := int64(15 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		respondError(w, http.StatusBadRequest, "Arquivo muito grande (máximo 15MB)")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read file from request")
		return
	}
	defer file.Close()

	// Validate file size
	if handler.Size > maxSize {
		respondError(w, http.StatusBadRequest, "Arquivo muito grande (máximo 15MB)")
		return
	}

	// Read file content for validation
	fileContent, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Erro ao ler arquivo")
		return
	}

	ext := strings.ToLower(filepath.Ext(handler.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" && ext != ".svg" {
		respondError(w, http.StatusBadRequest, "Tipo de arquivo inválido (permitidos: jpg, png, gif, webp, svg)")
		return
	}

	// Validate MIME type from content (not just header)
	detectedMime := http.DetectContentType(fileContent)

	// Special handling for SVG which might be detected as text/xml or text/plain
	isSVG := ext == ".svg" && (strings.Contains(detectedMime, "xml") || strings.Contains(detectedMime, "plain") || detectedMime == "image/svg+xml")

	if !allowedMimeTypes[detectedMime] && !isSVG {
		log.Printf("⚠️ Upload rejected: detected MIME type %s not allowed", detectedMime)
		respondError(w, http.StatusBadRequest, "Tipo de arquivo inválido detectado")
		return
	}

	// Validate magic bytes for additional security
	validMagic := false
	for mime, magic := range imageMagicBytes {
		if len(fileContent) >= len(magic) && bytes.HasPrefix(fileContent, magic) {
			validMagic = true
			// Special check for webp (RIFF header + WEBP)
			if mime == "image/webp" && len(fileContent) >= 12 {
				if string(fileContent[8:12]) != "WEBP" {
					validMagic = false
				}
			}
			break
		}
	}

	// For SVG, we check if it contains <svg somewhere near the beginning if it didn't match prefix
	if !validMagic && isSVG {
		// Check first 1024 bytes for <svg
		searchLen := 1024
		if len(fileContent) < searchLen {
			searchLen = len(fileContent)
		}
		if bytes.Contains(fileContent[:searchLen], []byte("<svg")) {
			validMagic = true
		}
	}

	if !validMagic {
		log.Printf("⚠️ Upload rejected: invalid magic bytes for file %s", handler.Filename)
		respondError(w, http.StatusBadRequest, "Arquivo corrompido ou tipo inválido")
		return
	}

	// Create uploads directory if not exists
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	// Generate unique filename with timestamp and random suffix
	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), generateRandomString(8), ext)
	dstPath := filepath.Join(uploadDir, filename)

	// Write file
	if err := os.WriteFile(dstPath, fileContent, 0644); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	// Return public URL
	publicUrl := fmt.Sprintf("/uploads/%s", filename)

	log.Printf("✅ User %s uploaded file: %s (%d bytes)", claims.Username, filename, len(fileContent))

	// Log audit
	if s.auditStore != nil {
		userAgent := r.UserAgent()
		method := r.Method
		path := r.URL.Path
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "upload",
			Resource:      "file",
			ResourceID:    filename,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			NewValue: map[string]interface{}{
				"filename":      filename,
				"original_name": handler.Filename,
				"size":          len(fileContent),
				"mime_type":     detectedMime,
			},
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     &userAgent,
			RequestMethod: &method,
			RequestPath:   &path,
		})
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"url": publicUrl,
	})
}

// generateRandomString generates a random alphanumeric string
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond) // Small delay for randomness
	}
	return string(b)
}
