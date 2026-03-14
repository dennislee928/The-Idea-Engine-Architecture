package main

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

// Message defines the payload sent through Kafka
type Message struct {
	PostID   string `json:"post_id"`
	Platform string `json:"platform"`
	Content  string `json:"content"`
	URL      string `json:"url"`
}

type KafkaQueue struct {
	writer *kafka.Writer
	reader *kafka.Reader
}

func NewKafkaQueue(brokerURL, topic string) *KafkaQueue {
	return &KafkaQueue{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokerURL),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{brokerURL},
			GroupID:  "idea-engine-llm-group",
			Topic:    topic,
			MaxBytes: 10e6, // 10MB
		}),
	}
}

func (k *KafkaQueue) Push(ctx context.Context, msg Message) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return k.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.PostID),
		Value: bytes,
	})
}

func (k *KafkaQueue) Consume(ctx context.Context) (Message, error) {
	m, err := k.reader.ReadMessage(ctx)
	if err != nil {
		return Message{}, err
	}
	var msg Message
	err = json.Unmarshal(m.Value, &msg)
	return msg, err
}
