package kiro

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Fingerprint 持有多维指纹数据，用于运行时请求伪装。
type Fingerprint struct {
	OIDCSDKVersion      string // 3.7xx (AWS SDK JS)
	RuntimeSDKVersion   string // 1.0.x (runtime API)
	StreamingSDKVersion string // 1.0.x (streaming API)
	OSType              string // darwin/windows/linux
	OSVersion           string
	NodeVersion         string
	KiroVersion         string
	KiroHash            string // SHA256
}

// FingerprintConfig 持有外部指纹配置覆盖项。
type FingerprintConfig struct {
	OIDCSDKVersion      string
	RuntimeSDKVersion   string
	StreamingSDKVersion string
	OSType              string
	OSVersion           string
	NodeVersion         string
	KiroVersion         string
	KiroHash            string
}

// FingerprintManager 管理每个账号的指纹生成和缓存。
type FingerprintManager struct {
	mu           sync.RWMutex
	fingerprints map[string]*Fingerprint // accountKey -> fingerprint
	rng          *rand.Rand
	config       *FingerprintConfig // 外部配置（可选）
}

var (
	// SDK 版本池
	oidcSDKVersions = []string{
		"3.980.0", "3.975.0", "3.972.0", "3.808.0",
		"3.738.0", "3.737.0", "3.736.0", "3.735.0",
	}
	// runtime API 使用的 SDK 版本
	runtimeSDKVersions = []string{"1.0.0"}
	// streaming API (generateAssistantResponse) 使用的 SDK 版本
	streamingSDKVersions = []string{"1.0.27"}
	// 有效的 OS 类型
	osTypes = []string{"darwin", "windows", "linux"}
	// OS 版本池
	osVersions = map[string][]string{
		"darwin":  {"25.2.0", "25.1.0", "25.0.0", "24.5.0", "24.4.0", "24.3.0"},
		"windows": {"10.0.26200", "10.0.26100", "10.0.22631", "10.0.22621", "10.0.19045"},
		"linux":   {"6.12.0", "6.11.0", "6.8.0", "6.6.0", "6.5.0", "6.1.0"},
	}
	// Node 版本池
	nodeVersions = []string{
		"22.21.1", "22.21.0", "22.20.0", "22.19.0", "22.18.0",
		"20.18.0", "20.17.0", "20.16.0",
	}
	// Kiro IDE 版本池
	kiroVersions = []string{
		"0.10.32", "0.10.16", "0.10.10",
		"0.9.47", "0.9.40", "0.9.2",
		"0.8.206", "0.8.140", "0.8.135", "0.8.86",
	}
	// 全局单例
	globalFingerprintManager     *FingerprintManager
	globalFingerprintManagerOnce sync.Once
)

// GlobalFingerprintManager 返回全局指纹管理器单例。
func GlobalFingerprintManager() *FingerprintManager {
	globalFingerprintManagerOnce.Do(func() {
		globalFingerprintManager = NewFingerprintManager()
	})
	return globalFingerprintManager
}

// SetGlobalFingerprintConfig 设置全局指纹配置。
func SetGlobalFingerprintConfig(cfg *FingerprintConfig) {
	GlobalFingerprintManager().SetConfig(cfg)
}

// SetConfig 应用配置并清除指纹缓存。
func (fm *FingerprintManager) SetConfig(cfg *FingerprintConfig) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.config = cfg
	// 清除缓存的指纹，使其使用新配置重新生成
	fm.fingerprints = make(map[string]*Fingerprint)
}

// NewFingerprintManager 创建新的指纹管理器。
func NewFingerprintManager() *FingerprintManager {
	return &FingerprintManager{
		fingerprints: make(map[string]*Fingerprint),
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetFingerprint 返回指定 accountKey 的指纹，如果不存在则创建一个。
func (fm *FingerprintManager) GetFingerprint(accountKey string) *Fingerprint {
	fm.mu.RLock()
	if fp, exists := fm.fingerprints[accountKey]; exists {
		fm.mu.RUnlock()
		return fp
	}
	fm.mu.RUnlock()

	fm.mu.Lock()
	defer fm.mu.Unlock()

	// 双重检查
	if fp, exists := fm.fingerprints[accountKey]; exists {
		return fp
	}

	fp := fm.generateFingerprint(accountKey)
	fm.fingerprints[accountKey] = fp
	return fp
}

func (fm *FingerprintManager) generateFingerprint(accountKey string) *Fingerprint {
	if fm.config != nil {
		return fm.generateFromConfig(accountKey)
	}
	return fm.generateRandom(accountKey)
}

// generateFromConfig 使用配置值生成指纹，空字段回退为随机值。
func (fm *FingerprintManager) generateFromConfig(accountKey string) *Fingerprint {
	cfg := fm.config

	configOrRandom := func(configVal string, choices []string) string {
		if configVal != "" {
			return configVal
		}
		return choices[fm.rng.Intn(len(choices))]
	}

	osType := cfg.OSType
	if osType == "" {
		osType = runtime.GOOS
		if !slices.Contains(osTypes, osType) {
			osType = osTypes[fm.rng.Intn(len(osTypes))]
		}
	}

	osVersion := cfg.OSVersion
	if osVersion == "" {
		if versions, ok := osVersions[osType]; ok {
			osVersion = versions[fm.rng.Intn(len(versions))]
		}
	}

	kiroHash := cfg.KiroHash
	if kiroHash == "" {
		hash := sha256.Sum256([]byte(accountKey))
		kiroHash = hex.EncodeToString(hash[:])
	}

	return &Fingerprint{
		OIDCSDKVersion:      configOrRandom(cfg.OIDCSDKVersion, oidcSDKVersions),
		RuntimeSDKVersion:   configOrRandom(cfg.RuntimeSDKVersion, runtimeSDKVersions),
		StreamingSDKVersion: configOrRandom(cfg.StreamingSDKVersion, streamingSDKVersions),
		OSType:              osType,
		OSVersion:           osVersion,
		NodeVersion:         configOrRandom(cfg.NodeVersion, nodeVersions),
		KiroVersion:         configOrRandom(cfg.KiroVersion, kiroVersions),
		KiroHash:            kiroHash,
	}
}

// generateRandom 基于 accountKey 哈希生成确定性随机指纹。
func (fm *FingerprintManager) generateRandom(accountKey string) *Fingerprint {
	hash := sha256.Sum256([]byte(accountKey))
	seed := int64(binary.BigEndian.Uint64(hash[:8]))
	rng := rand.New(rand.NewSource(seed))

	osType := runtime.GOOS
	if !slices.Contains(osTypes, osType) {
		osType = osTypes[rng.Intn(len(osTypes))]
	}
	osVersion := osVersions[osType][rng.Intn(len(osVersions[osType]))]

	return &Fingerprint{
		OIDCSDKVersion:      oidcSDKVersions[rng.Intn(len(oidcSDKVersions))],
		RuntimeSDKVersion:   runtimeSDKVersions[rng.Intn(len(runtimeSDKVersions))],
		StreamingSDKVersion: streamingSDKVersions[rng.Intn(len(streamingSDKVersions))],
		OSType:              osType,
		OSVersion:           osVersion,
		NodeVersion:         nodeVersions[rng.Intn(len(nodeVersions))],
		KiroVersion:         kiroVersions[rng.Intn(len(kiroVersions))],
		KiroHash:            hex.EncodeToString(hash[:]),
	}
}

// GenerateAccountKey 基于 SHA256(seed) 生成 16 字符的十六进制账号标识。
func GenerateAccountKey(seed string) string {
	hash := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(hash[:8])
}

// GetAccountKey 基于 clientID > refreshToken > 随机 UUID 派生账号标识。
func GetAccountKey(clientID, refreshToken string) string {
	if clientID != "" {
		return GenerateAccountKey(clientID)
	}
	if refreshToken != "" {
		return GenerateAccountKey(refreshToken)
	}
	return GenerateAccountKey(uuid.New().String())
}

// BuildUserAgent 构建 User-Agent header 值。
// 格式: aws-sdk-js/{SDKVersion} ua/2.1 os/{OSType}#{OSVersion} lang/js md/nodejs#{NodeVersion} api/codewhispererstreaming#{SDKVersion} m/E KiroIDE-{KiroVersion}-{KiroHash}
func (fp *Fingerprint) BuildUserAgent() string {
	return fmt.Sprintf(
		"aws-sdk-js/%s ua/2.1 os/%s#%s lang/js md/nodejs#%s api/codewhispererstreaming#%s m/E KiroIDE-%s-%s",
		fp.StreamingSDKVersion,
		fp.OSType,
		fp.OSVersion,
		fp.NodeVersion,
		fp.StreamingSDKVersion,
		fp.KiroVersion,
		fp.KiroHash,
	)
}

// BuildAmzUserAgent 构建 x-amz-user-agent header 值。
// 格式: aws-sdk-js/{SDKVersion} KiroIDE-{KiroVersion}-{KiroHash}
func (fp *Fingerprint) BuildAmzUserAgent() string {
	return fmt.Sprintf(
		"aws-sdk-js/%s KiroIDE-%s-%s",
		fp.StreamingSDKVersion,
		fp.KiroVersion,
		fp.KiroHash,
	)
}
