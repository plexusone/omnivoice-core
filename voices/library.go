// Package voices provides utilities for managing voice profiles.
package voices

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/omnivoice-core/config"
	"github.com/plexusone/omnivoice-core/tts"
)

// Profile represents a voice profile with its reference audio and metadata.
type Profile struct {
	// Slug is the directory name / profile ID
	Slug string `json:"slug"`

	// Name is the human-readable name
	Name string `json:"name,omitempty"`

	// Language is the BCP-47 language code
	Language string `json:"language,omitempty"`

	// Description is an optional description
	Description string `json:"description,omitempty"`

	// Dir is the full path to the profile directory
	Dir string `json:"-"`

	// ReferenceAudioPath is the path to reference.wav
	ReferenceAudioPath string `json:"-"`

	// ReferenceTextPath is the path to reference.txt
	ReferenceTextPath string `json:"-"`
}

// Metadata represents optional metadata for a voice profile.
type Metadata struct {
	Name        string   `json:"name,omitempty"`
	Language    string   `json:"language,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	Style       string   `json:"style,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Library manages a collection of voice profiles.
type Library struct {
	dir string
}

// NewLibrary creates a new voice library from the default or specified directory.
// If dir is empty, uses the default voices directory (~/.plexusone/omnivoice/voices).
func NewLibrary(dir string) *Library {
	if dir == "" {
		dir = config.GetVoicesDir()
	}
	return &Library{dir: dir}
}

// Dir returns the library directory path.
func (l *Library) Dir() string {
	return l.dir
}

// List returns all available voice profiles.
func (l *Library) List() ([]Profile, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Empty library
		}
		return nil, fmt.Errorf("failed to read voices directory: %w", err)
	}

	var profiles []Profile
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		profile, err := l.Get(e.Name())
		if err != nil {
			continue // Skip invalid profiles
		}
		profiles = append(profiles, *profile)
	}

	return profiles, nil
}

// Get returns a specific voice profile by slug.
func (l *Library) Get(slug string) (*Profile, error) {
	dir := filepath.Join(l.dir, slug)

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("voice profile not found: %s", slug)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", slug)
	}

	// Check for required files
	refAudioPath := filepath.Join(dir, "reference.wav")
	refTextPath := filepath.Join(dir, "reference.txt")

	if _, err := os.Stat(refAudioPath); err != nil {
		return nil, fmt.Errorf("reference.wav not found in %s", slug)
	}
	if _, err := os.Stat(refTextPath); err != nil {
		return nil, fmt.Errorf("reference.txt not found in %s", slug)
	}

	profile := &Profile{
		Slug:               slug,
		Dir:                dir,
		ReferenceAudioPath: refAudioPath,
		ReferenceTextPath:  refTextPath,
	}

	// Load optional metadata
	metadataPath := filepath.Join(dir, "metadata.json")
	if data, err := os.ReadFile(metadataPath); err == nil {
		var meta Metadata
		if json.Unmarshal(data, &meta) == nil {
			profile.Name = meta.Name
			profile.Language = meta.Language
			profile.Description = meta.Description
		}
	}

	// Use slug as name if not set
	if profile.Name == "" {
		profile.Name = slug
	}

	return profile, nil
}

// Exists checks if a voice profile exists.
func (l *Library) Exists(slug string) bool {
	_, err := l.Get(slug)
	return err == nil
}

// LoadReferenceAudio loads the reference audio bytes for a profile.
func (l *Library) LoadReferenceAudio(slug string) ([]byte, error) {
	profile, err := l.Get(slug)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(profile.ReferenceAudioPath)
}

// LoadReferenceText loads the reference transcript for a profile.
func (l *Library) LoadReferenceText(slug string) (string, error) {
	profile, err := l.Get(slug)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(profile.ReferenceTextPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PrepareProfile prepares a voice profile for use with a TTS provider.
// The provider must implement tts.ProfileCacher.
func (l *Library) PrepareProfile(ctx context.Context, slug string, cacher tts.ProfileCacher) error {
	refAudio, err := l.LoadReferenceAudio(slug)
	if err != nil {
		return err
	}

	refText, err := l.LoadReferenceText(slug)
	if err != nil {
		return err
	}

	profile, _ := l.Get(slug)

	_, err = cacher.PrepareVoiceProfile(ctx, tts.PrepareVoiceProfileRequest{
		ProfileID:      slug,
		ReferenceAudio: refAudio,
		ReferenceText:  refText,
		Language:       profile.Language,
	})

	return err
}

// Create creates a new voice profile directory with the given files.
func (l *Library) Create(slug string, referenceAudio []byte, referenceText string, meta *Metadata) error {
	dir := filepath.Join(l.dir, slug)

	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Write reference audio
	if err := os.WriteFile(filepath.Join(dir, "reference.wav"), referenceAudio, 0644); err != nil {
		return fmt.Errorf("failed to write reference.wav: %w", err)
	}

	// Write reference text
	if err := os.WriteFile(filepath.Join(dir, "reference.txt"), []byte(referenceText), 0644); err != nil {
		return fmt.Errorf("failed to write reference.txt: %w", err)
	}

	// Write metadata if provided
	if meta != nil {
		data, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0644); err != nil {
			return fmt.Errorf("failed to write metadata.json: %w", err)
		}
	}

	return nil
}

// Delete removes a voice profile.
func (l *Library) Delete(slug string) error {
	profile, err := l.Get(slug)
	if err != nil {
		return err
	}
	return os.RemoveAll(profile.Dir)
}
