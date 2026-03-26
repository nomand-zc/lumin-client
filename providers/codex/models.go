package codex

const (
	// GPT-5 系列模型
	GPT_5_1      = "gpt-5.1"
	GPT_5_1_MINI = "gpt-5.1-mini"
	GPT_5_1_NANO = "gpt-5.1-nano"
	GPT_5_2      = "gpt-5.2"

	// Codex 专用模型
	GPT_5_1_CODEX     = "gpt-5.1-codex"
	GPT_5_1_CODEX_MAX = "gpt-5.1-codex-max"
	GPT_5_2_CODEX     = "gpt-5.2-codex"

	// O 系列推理模型
	O3      = "o3"
	O3_PRO  = "o3-pro"
	O4_MINI = "o4-mini"
)

// ModelList 列出所有对外公开的模型名称
var (
	ModelList = []string{
		GPT_5_1,
		GPT_5_1_MINI,
		GPT_5_1_NANO,
		GPT_5_2,
		GPT_5_1_CODEX,
		GPT_5_1_CODEX_MAX,
		GPT_5_2_CODEX,
		O3,
		O3_PRO,
		O4_MINI,
	}

	// ModelMap 将对外模型名（包含别名）映射到内部实际使用的模型 ID
	ModelMap = map[string]string{
		GPT_5_1:           GPT_5_1,
		GPT_5_1_MINI:      GPT_5_1_MINI,
		GPT_5_1_NANO:      GPT_5_1_NANO,
		GPT_5_2:           GPT_5_2,
		GPT_5_1_CODEX:     GPT_5_1_CODEX,
		GPT_5_1_CODEX_MAX: GPT_5_1_CODEX_MAX,
		GPT_5_2_CODEX:     GPT_5_2_CODEX,
		O3:                O3,
		O3_PRO:            O3_PRO,
		O4_MINI:           O4_MINI,
	}
)
