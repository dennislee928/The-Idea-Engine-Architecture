package main

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

type KafkaQueue struct {
	writer *kafka.Writer
	reader *kafka.Reader
}

func NewKafkaQueue(brokerURL, topic, groupID string) *KafkaQueue {
	return &KafkaQueue{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokerURL),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{brokerURL},
			GroupID:  groupID,
			Topic:    topic,
			MaxBytes: 10e6,
		}),
	}
}

func (k *KafkaQueue) Push(ctx context.Context, msg QueueMessage) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return k.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.ID),
		Value: bytes,
	})
}

func (k *KafkaQueue) Consume(ctx context.Context) (QueueMessage, error) {
	m, err := k.reader.ReadMessage(ctx)
	if err != nil {
		return QueueMessage{}, err
	}
	var msg QueueMessage
	err = json.Unmarshal(m.Value, &msg)
	return msg, err
}

func (k *KafkaQueue) Close() error {
	if err := k.writer.Close(); err != nil {
		return err
	}
	return k.reader.Close()
}
