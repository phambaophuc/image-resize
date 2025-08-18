package queue

import (
	"fmt"

	"github.com/phambaophuc/image-resize/internal/services/processor"
	"github.com/phambaophuc/image-resize/internal/services/storage"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type QueueService struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	logger    *zap.Logger
	queueName string
	processor *processor.ImageProcessor
	storage   *storage.StorageService
}

func NewQueueService(
	rabbitmqURL string,
	processor *processor.ImageProcessor,
	storage *storage.StorageService,
	logger *zap.Logger,
) (*QueueService, error) {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	queueName := "image_processing"

	// Declare queue
	_, err = channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		conn.Close()
		channel.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &QueueService{
		conn:      conn,
		channel:   channel,
		logger:    logger,
		queueName: queueName,
		processor: processor,
		storage:   storage,
	}, nil
}

// Close closes the queue connection
func (q *QueueService) Close() error {
	if q.channel != nil {
		q.channel.Close()
	}
	if q.conn != nil {
		q.conn.Close()
	}
	return nil
}
