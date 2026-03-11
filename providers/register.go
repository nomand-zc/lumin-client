package providers

var registeredProviders = make(map[string]map[string]Provider)

// Register 注册一个 provider到全局管理器，可以作为单例使用
func Register(provider Provider) {
	providerType, providerName := provider.Type(), provider.Name()
	if registeredProviders[providerType] == nil {
		registeredProviders[providerType] = make(map[string]Provider)
	}
	registeredProviders[providerType][providerName] = provider
}

func GetProvider(providerType, providerName string) Provider {
	return registeredProviders[providerType][providerName]
}
