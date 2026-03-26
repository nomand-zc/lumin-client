package queue

import (
	"context"
	"sync/atomic"
)

type queue[T any] struct {
	c      chan T
	done   chan struct{} // 专用关闭信号，由 Close() 负责关闭
	closed uint32        // 使用原子操作保证并发安全
}

// New 创建一个基于channel的队列
func New[T any](bufferSize int) Queue[T] {
	return &queue[T]{
		c:    make(chan T, bufferSize),
		done: make(chan struct{}),
	}
}

// Len 返回队列中元素的个数
func (q *queue[T]) Len() int {
	return len(q.c)
}

// Pop 从队列中弹出一个元素，如果队列为空则阻塞等待。
// 队列关闭后仍可读取缓冲区中的剩余元素，直到缓冲区耗尽才返回 ErrQueueClosed。
// 通过双阶段 select 确保 done 触发后仍能正确排干缓冲区，并支持多消费者并发安全。
func (q *queue[T]) Pop(ctx context.Context) (T, error) {
	var zero T
	closed := false

	for {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case item := <-q.c:
			return item, nil
		case <-q.done:
			closed = true
		}

		if closed {
			// done 已触发，尝试非阻塞消费缓冲区中的剩余元素
			select {
			case item := <-q.c:
				return item, nil
			default:
				return zero, ErrQueueClosed
			}
		}
	}
}

// Each 阻塞式遍历队列中的所有元素，直到队列关闭或上下文取消。
func (q *queue[T]) Each(ctx context.Context, fn func(T) error) error {
	for {
		item, err := q.Pop(ctx)
		if err != nil {
			if !IsClosedError(err) {
				return err
			}
			break
		}
		if err := fn(item); err != nil {
			return err
		}
	}

	return nil
}

// Push 向队列中推入一个元素，如果队列已满则阻塞等待。
// 如果队列已关闭（包括在 Push 等待期间被关闭），返回 ErrQueueClosed。
// 通过同时监听独立的 done channel 和数据 channel，避免向已关闭 channel 发送而引发 panic。
func (q *queue[T]) Push(ctx context.Context, item T) error {
	// 使用原子操作快速检查关闭状态（fast path）
	if atomic.LoadUint32(&q.closed) == 1 {
		return ErrQueueClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-q.done:
		// done channel 已关闭，队列不再接受新数据
		return ErrQueueClosed
	case q.c <- item:
		return nil
	}
}

// Close 关闭队列。
// 通过关闭 done channel 通知所有阻塞的 Push/Pop goroutine，
// 不关闭数据 channel，避免"send on closed channel" panic。
func (q *queue[T]) Close() error {
	// 使用原子操作确保只关闭一次
	if atomic.CompareAndSwapUint32(&q.closed, 0, 1) {
		close(q.done)
		return nil
	}

	// 如果已经关闭，返回错误
	return ErrQueueClosed
}

// Closed 判断队列是否已关闭
func (q *queue[T]) Closed() bool {
	return atomic.LoadUint32(&q.closed) == 1
}
