package saramax

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"time"
	"webook/pkg/logger"
)

type BatchHandler[T any] struct {
	l  logger.LoggerV1
	fn func(msgs []*sarama.ConsumerMessage, ts []T) error
	// 用 option 模式来设置这个 batchSize 和 batchDuration
	batchSize     int
	batchDuration time.Duration
}

type Option[T any] func(*BatchHandler[T])

func WithBatchSize[T any](size int) Option[T] {
	return func(h *BatchHandler[T]) {
		h.batchSize = size
	}
}

func WithBatchDuration[T any](duration time.Duration) Option[T] {
	return func(h *BatchHandler[T]) {
		h.batchDuration = duration
	}
}

func NewBatchHandler[T any](l logger.LoggerV1,
	fn func(msgs []*sarama.ConsumerMessage, ts []T) error,
	opts ...Option[T]) *BatchHandler[T] {
	h := &BatchHandler[T]{
		l:             l,
		fn:            fn,
		batchSize:     10,          // 默认值
		batchDuration: time.Second, // 默认值
	}
	for _, opt := range opts {
		opt(h) // 应用配置
	}
	return h
}

func (b *BatchHandler[T]) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (b *BatchHandler[T]) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (b *BatchHandler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgsCh := claim.Messages()

	for {
		ctx, cancel := context.WithTimeout(session.Context(), b.batchDuration)
		msgs := make([]*sarama.ConsumerMessage, 0, b.batchSize)
		ts := make([]T, 0, b.batchSize)

		// 收集一批消息：最多 batchSize 条，或超时
		for i := 0; i < b.batchSize; i++ {
			select {
			case <-ctx.Done():
				// 超时，结束收集
				cancel()
				break
			case msg, ok := <-msgsCh:
				if !ok {
					cancel()
					if len(msgs) > 0 {
						b.processAndCommit(session, msgs, ts)
					}
					return nil
				}

				var t T
				if err := json.Unmarshal(msg.Value, &t); err != nil {
					b.l.Error("反序列化失败",
						logger.Error(err),
						logger.String("topic", msg.Topic),
						logger.Int32("partition", msg.Partition),
						logger.Int64("offset", msg.Offset))
					// 跳过坏消息（不加入批次）
					continue
				}
				msgs = append(msgs, msg)
				ts = append(ts, t)
			}
		}
		cancel()

		// 处理已收集的批次
		if len(msgs) > 0 {
			if err := b.processAndCommit(session, msgs, ts); err != nil {
				return err // 中断消费，触发重试
			}
		}
	}
}

// 仅在业务成功时提交 offset
func (b *BatchHandler[T]) processAndCommit(session sarama.ConsumerGroupSession,
	msgs []*sarama.ConsumerMessage, ts []T) error {
	if err := b.fn(msgs, ts); err != nil {
		b.l.Error("业务处理失败", logger.Error(err))
		return err
	}
	// 提交最后一条即可（Kafka 语义）
	session.MarkMessage(msgs[len(msgs)-1], "")
	return nil
}
