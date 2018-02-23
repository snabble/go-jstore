package jstore

// StoreProvider is a factory method for creation of stores.
type ProviderFunc func(connectionSource string, options ...StoreOption) (Store, error)

var provider = map[string]ProviderFunc{}

// RegisterProvider registers a factory method by the provider name.
func RegisterProvider(name string, providerFunc ProviderFunc) {
	provider[name] = providerFunc
}

// GetProvider returns a registered provider by its name.
// The bool return parameter indicated, if there was such a provider.
func getProvider(providerName string) (ProviderFunc, bool) {
	p, exist := provider[providerName]
	return p, exist
}

// ProviderList returns the names of all registered provider
func providerList() []string {
	list := make([]string, 0, len(provider))
	for k := range provider {
		list = append(list, k)
	}
	return list
}
