package chat

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ClipboardImage represents an image from clipboard
type ClipboardImage struct {
	Data   []byte
	Format string
}

// SaveImageToTemp saves an image to a temporary file and returns the path
func SaveImageToTemp(img *ClipboardImage) (string, error) {
	// Create temp directory if it doesn't exist
	tempDir := os.TempDir()
	cfTempDir := filepath.Join(tempDir, "codeforge-images")
	if err := os.MkdirAll(cfTempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	
	// Generate unique filename
	timestamp := time.Now().Format("20060102-150405")
	ext := ".png" // default to PNG
	if img.Format != "" {
		ext = "." + img.Format
	}
	filename := fmt.Sprintf("clipboard-%s%s", timestamp, ext)
	filepath := filepath.Join(cfTempDir, filename)
	
	// Write image data to file
	if err := os.WriteFile(filepath, img.Data, 0644); err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}
	
	return filepath, nil
}

// DecodeImage attempts to decode image data and determine format
func DecodeImage(data []byte) (*ClipboardImage, error) {
	reader := bytes.NewReader(data)
	_, format, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	return &ClipboardImage{
		Data:   data,
		Format: format,
	}, nil
}

// IsBase64Image checks if a string is a base64 encoded image
func IsBase64Image(s string) bool {
	// Check for data URL scheme
	if len(s) > 22 && s[:22] == "data:image/png;base64," {
		return true
	}
	if len(s) > 23 && s[:23] == "data:image/jpeg;base64," {
		return true
	}
	if len(s) > 22 && s[:22] == "data:image/gif;base64," {
		return true
	}
	return false
}

// DecodeBase64Image decodes a base64 image string
func DecodeBase64Image(s string) (*ClipboardImage, error) {
	var data string
	var format string
	
	// Extract format and data from data URL
	if len(s) > 22 && s[:22] == "data:image/png;base64," {
		data = s[22:]
		format = "png"
	} else if len(s) > 23 && s[:23] == "data:image/jpeg;base64," {
		data = s[23:]
		format = "jpeg"
	} else if len(s) > 22 && s[:22] == "data:image/gif;base64," {
		data = s[22:]
		format = "gif"
	} else {
		return nil, fmt.Errorf("unsupported image format")
	}
	
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	
	return &ClipboardImage{
		Data:   decoded,
		Format: format,
	}, nil
}

// GetImageFromReader reads image data from an io.Reader
func GetImageFromReader(r io.Reader) (*ClipboardImage, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	
	return DecodeImage(data)
}