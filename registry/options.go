package registry

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
