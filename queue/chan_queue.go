package queue

import (
	"context"
	"sync/atomic"
)

type queue[T any] struct {
	c      chan T
	closed uint32 // 使用原子操作保证并发安全
}

// New 创建一个基于channel的队列
func New[T any](bufferSize int) Queue[T] {
	return &queue[T]{
		c: make(chan T, bufferSize),
	}
}

// Len 返回队列中元素的个数
func (q *queue[T]) Len() int {
	return len(q.c)
}

// Pop 从队列中弹出一个元素，如果队列为空则阻塞等待
func (q *queue[T]) Pop(ctx context.Context) (T, error) {
	var zero T

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case item, ok := <-q.c:
		if !ok {
			return zero, ErrQueueClosed
		}
		return item, nil
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

// Push 向队列中推入一个元素，如果队列已满则阻塞等待
func (q *queue[T]) Push(ctx context.Context, item T) error {
	// 使用原子操作快速检查关闭状态
	if atomic.LoadUint32(&q.closed) == 1 {
		return ErrQueueClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.c <- item:
		return nil
	}
}

// Close 关闭队列
func (q *queue[T]) Close() error {

	// 使用原子操作确保只关闭一次
	if atomic.CompareAndSwapUint32(&q.closed, 0, 1) {
		close(q.c)
		return nil
	}

	// 如果已经关闭，返回错误
	return ErrQueueClosed
}

// Closed 判断队列是否已关闭
func (q *queue[T]) Closed() bool {
	return atomic.LoadUint32(&q.closed) == 1
}
