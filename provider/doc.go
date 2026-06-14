// Package provider provides generic multi-provider client management.
//
// The Client[T] type provides common functionality for managing multiple
// providers with primary/fallback selection. It is designed to be embedded
// in domain-specific clients (tts.Client, stt.Client, realtime.Client).
//
// Example usage:
//
//	type MyClient struct {
//	    *provider.Client[MyProvider]
//	    hook MyHook
//	}
//
//	func NewMyClient(providers ...MyProvider) *MyClient {
//	    return &MyClient{
//	        Client: provider.NewClient(providers...),
//	    }
//	}
package provider
