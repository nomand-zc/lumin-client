package queue

import (
	"context"
	"sync"
	"testing"
)

// TestPushAfterCloseNoPanic 验证在 Close() 和 Push() 并发场景下不会 panic
// 这个测试用 -race 标志运行可以检测数据竞态
func TestPushAfterCloseNoPanic(t *testing.T) {
	for i := 0; i < 100; i++ {
		q := New[int](1)
		var wg sync.WaitGroup

		// 并发关闭
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Close()
		}()

		// 并发 Push，可能在 Close 前或后执行
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 使用 recover 保护调用方，验证 Push 内部不会 panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Push panicked: %v", r)
				}
			}()
			_ = q.Push(context.Background(), 42)
		}()

		wg.Wait()
	}
}

// TestPushOnClosedChannelNoPanic 验证向已关闭 channel 的 Push 返回 ErrQueueClosed 而非 panic
func TestPushOnClosedChannelNoPanic(t *testing.T) {
	q := New[int](0) // 无缓冲队列，Push 必须等待 Pop

	// 先关闭队列
	if err := q.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Push 到已关闭的队列应该返回 ErrQueueClosed，不应该 panic
	err := q.Push(context.Background(), 42)
	if err == nil {
		t.Error("Push to closed queue should return error")
	}
	if !IsClosedError(err) {
		t.Errorf("Expected ErrQueueClosed, got %v", err)
	}
}

// TestPopDrainBufferAfterClose 验证 Close 后多个并发消费者能正确排干缓冲区，数据不丢失
func TestPopDrainBufferAfterClose(t *testing.T) {
	const bufferSize = 10
	const numItems = 10
	const numConsumers = 5

	q := New[int](bufferSize)

	// 先填满缓冲区
	for i := 0; i < numItems; i++ {
		if err := q.Push(context.Background(), i); err != nil {
			t.Fatalf("Push failed: %v", err)
		}
	}

	// 关闭队列
	if err := q.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 多个消费者并发排干缓冲区
	var mu sync.Mutex
	received := make([]int, 0, numItems)
	var wg sync.WaitGroup

	for i := 0; i < numConsumers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				item, err := q.Pop(context.Background())
				if err != nil {
					if IsClosedError(err) {
						return
					}
					t.Errorf("Unexpected error: %v", err)
					return
				}
				mu.Lock()
				received = append(received, item)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// 验证所有数据都被消费，没有丢失
	if len(received) != numItems {
		t.Errorf("Expected %d items consumed, got %d (data loss detected)", numItems, len(received))
	}
}

// TestConcurrentPushAndClose 大量并发 Push 与 Close 混合，验证无 panic
func TestConcurrentPushAndClose(t *testing.T) {
	for iteration := 0; iteration < 50; iteration++ {
		q := New[int](10)
		var wg sync.WaitGroup
		panicCh := make(chan interface{}, 20)

		// 启动多个 goroutine 并发 Push
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCh <- r
					}
				}()
				_ = q.Push(context.Background(), val)
			}(i)
		}

		// 同时关闭队列
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = q.Close()
		}()

		wg.Wait()
		close(panicCh)

		for p := range panicCh {
			t.Errorf("goroutine panicked during concurrent Push/Close: %v", p)
		}
	}
}
