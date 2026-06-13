package bargein

import "time"

// InterruptionMode defines when barge-in should trigger.
type InterruptionMode string

const (
	// ModeImmediate stops agent speech immediately when user speech is detected.
	// This provides the most responsive experience but may trigger on brief noises.
	ModeImmediate InterruptionMode = "immediate"

	// ModeAfterSentence waits for the agent to complete the current sentence
	// before stopping. This provides a smoother experience but higher latency.
	ModeAfterSentence InterruptionMode = "after_sentence"

	// ModeDisabled disables barge-in entirely. The user must wait for the
	// agent to finish speaking before their speech is processed.
	ModeDisabled InterruptionMode = "disabled"
)

// Config configures the barge-in detector.
type Config struct {
	// Mode determines when to trigger barge-in.
	// Default: ModeImmediate
	Mode InterruptionMode

	// MinSpeechDurationMs is the minimum speech duration in milliseconds
	// before triggering barge-in. This prevents false triggers from brief
	// noises or non-speech sounds.
	// Default: 200ms
	MinSpeechDurationMs int

	// SilenceThresholdMs is the duration of silence in milliseconds that
	// indicates the user has stopped speaking. Used in ModeAfterSentence.
	// Default: 500ms
	SilenceThresholdMs int

	// CooldownMs is the minimum time in milliseconds between consecutive
	// barge-in events. Prevents rapid repeated triggering.
	// Default: 300ms
	CooldownMs int

	// MinAgentSpeechMs is the minimum agent speech duration in milliseconds
	// before barge-in is enabled. Prevents interrupting very short responses.
	// Default: 500ms
	MinAgentSpeechMs int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Mode:                ModeImmediate,
		MinSpeechDurationMs: 200,
		SilenceThresholdMs:  500,
		CooldownMs:          300,
		MinAgentSpeechMs:    500,
	}
}

// Validate checks if the config is valid and applies defaults.
// Negative values are replaced with defaults. Zero values are allowed
// and represent "no minimum" for the respective setting.
func (c *Config) Validate() Config {
	result := *c

	if result.Mode == "" {
		result.Mode = ModeImmediate
	}
	if result.MinSpeechDurationMs < 0 {
		result.MinSpeechDurationMs = 200
	}
	if result.SilenceThresholdMs < 0 {
		result.SilenceThresholdMs = 500
	}
	if result.CooldownMs < 0 {
		result.CooldownMs = 300
	}
	if result.MinAgentSpeechMs < 0 {
		result.MinAgentSpeechMs = 500
	}

	return result
}

// MinSpeechDuration returns MinSpeechDurationMs as a time.Duration.
func (c Config) MinSpeechDuration() time.Duration {
	return time.Duration(c.MinSpeechDurationMs) * time.Millisecond
}

// SilenceThreshold returns SilenceThresholdMs as a time.Duration.
func (c Config) SilenceThreshold() time.Duration {
	return time.Duration(c.SilenceThresholdMs) * time.Millisecond
}

// Cooldown returns CooldownMs as a time.Duration.
func (c Config) Cooldown() time.Duration {
	return time.Duration(c.CooldownMs) * time.Millisecond
}

// MinAgentSpeech returns MinAgentSpeechMs as a time.Duration.
func (c Config) MinAgentSpeech() time.Duration {
	return time.Duration(c.MinAgentSpeechMs) * time.Millisecond
}
