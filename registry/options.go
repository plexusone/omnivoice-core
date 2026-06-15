package registry

import (
	"log/slog"
	"net"
	"time"
)

// WithAPIKey sets the API key for the provider.
func WithAPIKey(apiKey string) ProviderOption {
	return func(c *ProviderConfig) {
		c.APIKey = apiKey
	}
}

// WithBaseURL sets a custom base URL for the provider.
func WithBaseURL(baseURL string) ProviderOption {
	return func(c *ProviderConfig) {
		c.BaseURL = baseURL
	}
}

// WithExtension sets a provider-specific configuration value.
func WithExtension(key string, value any) ProviderOption {
	return func(c *ProviderConfig) {
		if c.Extensions == nil {
			c.Extensions = make(map[string]any)
		}
		c.Extensions[key] = value
	}
}

// CallSystem-specific option functions.
// These are convenience wrappers that set Extension values with standardized keys.

// WithAccountSID sets the account SID (Twilio).
func WithAccountSID(sid string) ProviderOption {
	return WithExtension("accountSID", sid)
}

// WithAuthToken sets the auth token (Twilio).
func WithAuthToken(token string) ProviderOption {
	return WithExtension("authToken", token)
}

// WithPhoneNumber sets the default outbound phone number.
func WithPhoneNumber(number string) ProviderOption {
	return WithExtension("phoneNumber", number)
}

// WithWebhookURL sets the webhook URL for incoming calls.
func WithWebhookURL(url string) ProviderOption {
	return WithExtension("webhookURL", url)
}

// WithRegion sets the service region.
func WithRegion(region string) ProviderOption {
	return WithExtension("region", region)
}

// Gateway-specific option functions.

// WithListener sets an external net.Listener for the gateway server.
// Useful for ngrok or custom listeners.
func WithListener(listener net.Listener) ProviderOption {
	return WithExtension("listener", listener)
}

// WithPublicURL sets the public URL for webhooks.
func WithPublicURL(url string) ProviderOption {
	return WithExtension("publicURL", url)
}

// WithListenAddr sets the address for the gateway to listen on.
func WithListenAddr(addr string) ProviderOption {
	return WithExtension("listenAddr", addr)
}

// WithConnectionID sets the connection ID (Telnyx).
func WithConnectionID(id string) ProviderOption {
	return WithExtension("connectionID", id)
}

// Realtime-specific option functions.

// WithVoice sets the voice for realtime audio output.
func WithVoice(voice string) ProviderOption {
	return WithExtension("voice", voice)
}

// WithModel sets the model for the provider.
func WithModel(model string) ProviderOption {
	return WithExtension("model", model)
}

// WithInstructions sets the system prompt/instructions.
func WithInstructions(instructions string) ProviderOption {
	return WithExtension("instructions", instructions)
}

// Pipeline configuration options.

// WithSTTProvider sets the STT provider name.
func WithSTTProvider(provider string) ProviderOption {
	return WithExtension("sttProvider", provider)
}

// WithSTTAPIKey sets the STT API key.
func WithSTTAPIKey(apiKey string) ProviderOption {
	return WithExtension("sttAPIKey", apiKey)
}

// WithSTTModel sets the STT model.
func WithSTTModel(model string) ProviderOption {
	return WithExtension("sttModel", model)
}

// WithSTTLanguage sets the STT language.
func WithSTTLanguage(language string) ProviderOption {
	return WithExtension("sttLanguage", language)
}

// WithTTSProvider sets the TTS provider name.
func WithTTSProvider(provider string) ProviderOption {
	return WithExtension("ttsProvider", provider)
}

// WithTTSAPIKey sets the TTS API key.
func WithTTSAPIKey(apiKey string) ProviderOption {
	return WithExtension("ttsAPIKey", apiKey)
}

// WithTTSVoiceID sets the TTS voice ID.
func WithTTSVoiceID(voiceID string) ProviderOption {
	return WithExtension("ttsVoiceID", voiceID)
}

// WithTTSModel sets the TTS model.
func WithTTSModel(model string) ProviderOption {
	return WithExtension("ttsModel", model)
}

// WithLLMProvider sets the LLM provider name.
func WithLLMProvider(provider string) ProviderOption {
	return WithExtension("llmProvider", provider)
}

// WithLLMAPIKey sets the LLM API key.
func WithLLMAPIKey(apiKey string) ProviderOption {
	return WithExtension("llmAPIKey", apiKey)
}

// WithLLMModel sets the LLM model.
func WithLLMModel(model string) ProviderOption {
	return WithExtension("llmModel", model)
}

// WithLLMSystemPrompt sets the LLM system prompt.
func WithLLMSystemPrompt(prompt string) ProviderOption {
	return WithExtension("llmSystemPrompt", prompt)
}

// Session configuration options.

// WithGreeting sets the initial greeting message.
func WithGreeting(greeting string) ProviderOption {
	return WithExtension("greeting", greeting)
}

// WithMaxSessionDuration sets the maximum session duration.
func WithMaxSessionDuration(d time.Duration) ProviderOption {
	return WithExtension("maxSessionDuration", d)
}

// WithInterruptionMode sets the interruption mode.
func WithInterruptionMode(mode string) ProviderOption {
	return WithExtension("interruptionMode", mode)
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) ProviderOption {
	return WithExtension("logger", logger)
}

// WithPipelineMode sets the pipeline mode ("text" or "realtime").
func WithPipelineMode(mode string) ProviderOption {
	return WithExtension("pipelineMode", mode)
}
