package utils

import (
	"context"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"netops/conf"
	"time"
)

type kafkaHandler struct {
	topic            string
	key              string
	bootstrapServers []string
	writer           *kafka.Writer
	Err              error
}

func NewKafkaHandlerDefault() *kafkaHandler {
	return NewKafkaHandler(conf.Config.Kafka.Topic, conf.Config.Kafka.Key, conf.Config.Kafka.BootstrapServers)
}

func NewKafkaHandler(topic, key string, bootstrapServer []string) *kafkaHandler {
	k := &kafkaHandler{topic: topic, key: key, bootstrapServers: bootstrapServer}
	k.init()
	return k
}

func (h *kafkaHandler) init() {
	h.writer = &kafka.Writer{
		Addr:                   kafka.TCP(h.bootstrapServers...),
		Topic:                  h.topic,
		AllowAutoTopicCreation: true,
	}
}

func (h *kafkaHandler) Producer(value []byte, headerInfo map[string]string) error {
	if h.Err != nil {
		return h.Err
	}
	l := zap.L().With(zap.String("func", "kafka Producer"), zap.String("topic", h.topic), zap.Strings("servers", h.bootstrapServers))
	l.Info("发送kafka消息", zap.ByteString("value", value), zap.Any("headers", headerInfo))

	headers := make([]kafka.Header, 0)
	for k, v := range headerInfo {
		headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	messages := []kafka.Message{
		{
			Key:     []byte(h.key),
			Value:   value,
			Headers: headers,
		},
	}

	var err error
	const retries = 3
	for i := 0; i < retries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// attempt to create topic prior to publishing the message
		err = h.writer.WriteMessages(ctx, messages...)
		if errors.Is(err, kafka.LeaderNotAvailable) || errors.Is(err, context.DeadlineExceeded) {
			time.Sleep(time.Millisecond * 250)
			continue
		}

		if err != nil {
			l.Error("消息发送失败", zap.Error(err))
			return fmt.Errorf("发送kafka失败, err: %w", err)
		}
		l.Info("消息发送成功")
		break
	}

	if e := h.writer.Close(); e != nil {
		l.Error(fmt.Sprintf("failed to close writer. %s", e.Error()))
	}
	return nil
}
