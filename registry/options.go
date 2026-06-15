package registry

import "net"

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
