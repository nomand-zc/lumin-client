package providers

const (
	DefaultProviderName = "default"
)

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
	if providers, ok := registeredProviders[providerType]; ok {
		if p, ok := providers[providerName]; ok {
			return p
		}
		// 未找到指定名称的 Provider，尝试返回 "default"
		return providers[DefaultProviderName]
	}
	return nil
}
