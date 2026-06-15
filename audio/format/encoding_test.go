package format

import "testing"

func TestEncoding_Normalize(t *testing.T) {
	tests := []struct {
		input Encoding
		want  Encoding
	}{
		// Linear16 variants
		{"linear16", Linear16},
		{"LINEAR16", Linear16},
		{" linear16 ", Linear16},
		{"pcm16", Linear16},
		{"PCM16", Linear16},
		{"pcm", Linear16},
		{"pcm_s16le", Linear16},
		{"wav", Linear16}, // WAV container → raw PCM

		// MuLaw variants
		{"mulaw", MuLaw},
		{"MULAW", MuLaw},
		{" mulaw ", MuLaw},
		{"ulaw", MuLaw},
		{"g711u", MuLaw},
		{"pcm_mulaw", MuLaw},

		// ALaw variants
		{"alaw", ALaw},
		{"ALAW", ALaw},
		{" alaw ", ALaw},
		{"g711a", ALaw},
		{"pcm_alaw", ALaw},

		// Container formats
		{"mp3", MP3},
		{"MP3", MP3},
		{"opus", Opus},
		{"OPUS", Opus},
		{"flac", FLAC},
		{"aac", AAC},
		{"speex", Speex},
		{"SPEEX", Speex},
		{"webm", WebM},
		{"WEBM", WebM},

		// Unknown passthrough
		{"unknown", Encoding("unknown")},
		{"ogg", Encoding("ogg")},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := tt.input.Normalize()
			if got != tt.want {
				t.Errorf("Encoding(%q).Normalize() = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncoding_IsRaw(t *testing.T) {
	tests := []struct {
		encoding Encoding
		want     bool
	}{
		// Raw formats
		{Linear16, true},
		{MuLaw, true},
		{ALaw, true},

		// Raw format variants (should normalize first)
		{"pcm16", true},
		{"ulaw", true},
		{"g711a", true},
		{" LINEAR16 ", true},
		{"wav", true}, // WAV normalizes to linear16

		// Container formats
		{MP3, false},
		{Opus, false},
		{FLAC, false},
		{AAC, false},
		{Speex, false},
		{WebM, false},

		// Unknown
		{"ogg", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.encoding), func(t *testing.T) {
			got := tt.encoding.IsRaw()
			if got != tt.want {
				t.Errorf("Encoding(%q).IsRaw() = %v, want %v", tt.encoding, got, tt.want)
			}
		})
	}
}

func TestIsRawEncoding(t *testing.T) {
	tests := []struct {
		encoding string
		want     bool
	}{
		// Raw formats
		{"linear16", true},
		{"LINEAR16", true},
		{" linear16 ", true},
		{"pcm16", true},
		{"mulaw", true},
		{"ulaw", true},
		{"alaw", true},
		{"g711u", true},
		{"g711a", true},
		{"wav", true}, // WAV normalizes to linear16

		// Container formats
		{"mp3", false},
		{"opus", false},
		{"flac", false},
		{"aac", false},
		{"speex", false},
		{"webm", false},

		// Edge cases
		{"", false},
		{"   ", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.encoding, func(t *testing.T) {
			got := IsRawEncoding(tt.encoding)
			if got != tt.want {
				t.Errorf("IsRawEncoding(%q) = %v, want %v", tt.encoding, got, tt.want)
			}
		})
	}
}

func TestEncoding_String(t *testing.T) {
	tests := []struct {
		encoding Encoding
		want     string
	}{
		{Linear16, "linear16"},
		{MuLaw, "mulaw"},
		{ALaw, "alaw"},
		{MP3, "mp3"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.encoding.String()
			if got != tt.want {
				t.Errorf("Encoding.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
