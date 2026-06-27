// Package config provides configuration constants and utilities for omnivoice-core.
package config

import (
	"os"
	"path/filepath"
)

const (
	// PlexusOneDir is the base directory for PlexusOne data.
	PlexusOneDir = ".plexusone"

	// OmniVoiceDir is the subdirectory for OmniVoice data.
	OmniVoiceDir = "omnivoice"

	// VoicesDir is the subdirectory for voice profiles.
	VoicesDir = "voices"
)

// DefaultVoicesDir returns the default voice profiles directory.
// Returns ~/.plexusone/omnivoice/voices
func DefaultVoicesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, PlexusOneDir, OmniVoiceDir, VoicesDir)
}

// VoicesDir returns the voice profiles directory.
// Uses OMNIVOICE_VOICES_DIR environment variable if set,
// otherwise returns the default path.
func GetVoicesDir() string {
	if dir := os.Getenv("OMNIVOICE_VOICES_DIR"); dir != "" {
		return dir
	}
	return DefaultVoicesDir()
}

// VoiceProfileDir returns the directory for a specific voice profile.
func VoiceProfileDir(profileSlug string) string {
	return filepath.Join(GetVoicesDir(), profileSlug)
}

// DefaultDataDir returns the default OmniVoice data directory.
// Returns ~/.plexusone/omnivoice
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, PlexusOneDir, OmniVoiceDir)
}

// EnsureVoicesDir creates the voices directory if it doesn't exist.
func EnsureVoicesDir() error {
	dir := GetVoicesDir()
	return os.MkdirAll(dir, 0755)
}

// EnsureVoiceProfileDir creates a voice profile directory if it doesn't exist.
func EnsureVoiceProfileDir(profileSlug string) error {
	dir := VoiceProfileDir(profileSlug)
	return os.MkdirAll(dir, 0755)
}
