package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestChainQueue(t *testing.T) {
	// 测试创建队列
	q := New[int](10)

	// 测试Closed方法
	if q.Closed() {
		t.Error("New queue should not be closed")
	}

	// 测试Len方法 - 空队列
	if q.Len() != 0 {
		t.Errorf("Empty queue should have length 0, got %d", q.Len())
	}

	// 测试Push方法
	err := q.Push(context.Background(), 42)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// 测试Len方法 - 推入后
	if q.Len() != 1 {
		t.Errorf("Queue should have length 1 after push, got %d", q.Len())
	}

	// 测试Pop方法
	item, err := q.Pop(context.Background())
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
	if item != 42 {
		t.Errorf("Expected 42, got %d", item)
	}

	// 测试Len方法 - 弹出后
	if q.Len() != 0 {
		t.Errorf("Queue should have length 0 after pop, got %d", q.Len())
	}

	// 测试批量推入
	for i := range 5 {
		err = q.Push(context.Background(), i)
		if err != nil {
			t.Errorf("Push %d failed: %v", i, err)
		}
	}

	// 测试Len方法 - 批量推入后
	if q.Len() != 5 {
		t.Errorf("Queue should have length 5 after batch push, got %d", q.Len())
	}

	// 测试Close方法
	err = q.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// 测试关闭后状态
	if !q.Closed() {
		t.Error("Queue should be closed after Close()")
	}

	// 测试关闭后推入
	err = q.Push(context.Background(), 100)
	if err == nil {
		t.Error("Push should fail on closed queue")
	}

	// 测试关闭后弹出 - 队列中还有数据，应该能成功弹出
	for i := range 5 {
		item, err = q.Pop(context.Background())
		if err != nil {
			t.Errorf("Pop should succeed on closed queue with data, got error: %v", err)
		}
		if item != i {
			t.Errorf("Expected %d, got %d", i, item)
		}
	}

	// 测试Len方法 - 读取所有数据后
	if q.Len() != 0 {
		t.Errorf("Queue should have length 0 after reading all data, got %d", q.Len())
	}

	// 测试关闭后弹出空队列
	_, err = q.Pop(context.Background())
	if err == nil {
		t.Error("Pop should fail on closed empty queue")
	}

	fmt.Println("All queue tests passed!")
}

// TestConcurrentAccess 测试并发读写场景
func TestConcurrentAccess(t *testing.T) {
	q := New[int](100)
	var wg sync.WaitGroup

	// 启动10个写入goroutine
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				err := q.Push(context.Background(), id*100+j)
				if err != nil && !IsClosedError(err) && !IsFullError(err) {
					t.Errorf("Unexpected error from goroutine %d: %v", id, err)
				}
				// 如果是队列满错误，稍等再试
				if IsFullError(err) {
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}

	// 启动10个读取goroutine
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, err := q.Pop(context.Background())
				if err != nil && !IsClosedError(err) {
					t.Errorf("Pop error from goroutine %d: %v", id, err)
				}
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 验证队列状态
	if q.Len() != 0 {
		t.Errorf("Queue should be empty after concurrent access, got length %d", q.Len())
	}

	fmt.Println("Concurrent access test passed!")
}

// TestConcurrentClose 测试并发关闭场景
func TestConcurrentClose(t *testing.T) {
	q := New[int](10)
	var wg sync.WaitGroup

	// 启动多个goroutine同时尝试关闭
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := q.Close()
			if err != nil && !IsClosedError(err) {
				t.Errorf("Close error from goroutine %d: %v", id, err)
			}
		}(i)
	}

	// 同时启动读写goroutine
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				err := q.Push(context.Background(), id*10+j)
				if err != nil && !IsClosedError(err) && !IsFullError(err) {
					t.Errorf("Push error: %v", err)
				}

				_, err = q.Pop(context.Background())
				if err != nil && !IsClosedError(err) {
					t.Errorf("Pop error: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证队列已关闭
	if !q.Closed() {
		t.Error("Queue should be closed after concurrent close attempts")
	}

	fmt.Println("Concurrent close test passed!")
}

// TestQueueFull 测试队列满的情况
func TestQueueFull(t *testing.T) {
	q := New[int](5) // 小缓冲区

	// 填满队列
	for i := 0; i < 5; i++ {
		err := q.Push(context.Background(), i)
		if err != nil {
			t.Errorf("Push failed: %v", err)
		}
	}

	// 尝试推入第6个元素，使用短超时的context，队列满时应因超时返回错误
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := q.Push(ctx, 6)
	if err == nil {
		t.Error("Push should fail on full queue with timeout context")
	}

	fmt.Println("Queue full test passed!")
}
