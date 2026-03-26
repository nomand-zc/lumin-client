package sse

import (
	"regexp"
	"sync"
	"testing"
)

// TestRetryAfterRegexConcurrency 验证 retryAfterRegex() 在并发调用下无数据竞态
// 用 go test -race 运行可检测竞态
func TestRetryAfterRegexConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	regexResults := make([]*regexp.Regexp, 0, 100)
	regexMutex := sync.Mutex{}

	// 并发调用 retryAfterRegex() 多次
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个 goroutine 调用多次 retryAfterRegex()，应该返回相同的指针
			regex1 := retryAfterRegex()
			regex2 := retryAfterRegex()

			regexMutex.Lock()
			regexResults = append(regexResults, regex1, regex2)
			regexMutex.Unlock()
		}()
	}

	wg.Wait()

	// 验证所有返回的 regex 都是同一个对象
	if len(regexResults) > 0 {
		firstRegex := regexResults[0]
		for i, r := range regexResults[1:] {
			if r != firstRegex {
				t.Errorf("retryAfterRegex() should return same pointer for all calls, but regexResults[%d] != regexResults[0]", i+1)
			}
		}
	}
}

// TestRetryAfterRegexPatternMatch 验证 retryAfterRegex 正则表达式模式正确
func TestRetryAfterRegexPatternMatch(t *testing.T) {
	regex := retryAfterRegex()

	tests := []struct {
		name    string
		text    string
		matches bool
	}{
		{
			name:    "整数秒数",
			text:    "try again in 30 s",
			matches: true,
		},
		{
			name:    "浮点秒数",
			text:    "try again in 30.5 seconds",
			matches: true,
		},
		{
			name:    "毫秒",
			text:    "try again in 500 ms",
			matches: true,
		},
		{
			name:    "大小写不敏感",
			text:    "TRY AGAIN IN 10 S",
			matches: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := regex.MatchString(tt.text)
			if match != tt.matches {
				t.Errorf("Expected match=%v, got %v for text: %s", tt.matches, match, tt.text)
			}
		})
	}
}
