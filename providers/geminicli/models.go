package geminicli

const (
	// Gemini 2.5 系列模型
	GEMINI_2_5_PRO        = "gemini-2.5-pro"
	GEMINI_2_5_FLASH      = "gemini-2.5-flash"
	GEMINI_2_5_FLASH_LITE = "gemini-2.5-flash-lite"

	// Gemini 2.0 系列模型
	GEMINI_2_0_FLASH      = "gemini-2.0-flash"
	GEMINI_2_0_FLASH_LITE = "gemini-2.0-flash-lite"
)

// ModelList 列出所有对外公开的模型名称（包含别名）
var (
	ModelList = []string{
		GEMINI_2_5_PRO,
		GEMINI_2_5_FLASH,
		GEMINI_2_5_FLASH_LITE,
		GEMINI_2_0_FLASH,
		GEMINI_2_0_FLASH_LITE,
		// 带日期版本的别名
		"gemini-2.5-pro-preview-05-06",
		"gemini-2.5-flash-preview-04-17",
	}

	// ModelMap 将对外模型名（包含别名）映射到内部实际使用的模型 ID
	ModelMap = map[string]string{
		GEMINI_2_5_PRO:        GEMINI_2_5_PRO,
		GEMINI_2_5_FLASH:      GEMINI_2_5_FLASH,
		GEMINI_2_5_FLASH_LITE: GEMINI_2_5_FLASH_LITE,
		GEMINI_2_0_FLASH:      GEMINI_2_0_FLASH,
		GEMINI_2_0_FLASH_LITE: GEMINI_2_0_FLASH_LITE,

		// 带日期版本的别名
		"gemini-2.5-pro-preview-05-06":   GEMINI_2_5_PRO,
		"gemini-2.5-flash-preview-04-17": GEMINI_2_5_FLASH,
	}
)
