package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/phambaophuc/image-resizing/internal/models"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type QueueService struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	logger    *zap.Logger
	queueName string
	processor *ImageProcessor
	storage   *StorageService
}

func NewQueueService(rabbitmqURL string, processor *ImageProcessor, storage *StorageService, logger *zap.Logger) (*QueueService, error) {
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

// PublishJob publishes a processing job to the queue
func (q *QueueService) PublishJob(ctx context.Context, job *models.ProcessingJob) error {
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	err = q.channel.Publish(
		"",          // exchange
		q.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         jobBytes,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	q.logger.Info("Job published to queue", zap.String("job_id", job.ID))
	return nil
}

// StartWorker starts consuming jobs from the queue
func (q *QueueService) StartWorker(ctx context.Context, workerID int) error {
	msgs, err := q.channel.Consume(
		q.queueName,                        // queue
		fmt.Sprintf("worker-%d", workerID), // consumer
		false,                              // auto-ack
		false,                              // exclusive
		false,                              // no-local
		false,                              // no-wait
		nil,                                // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	q.logger.Info("Worker started", zap.Int("worker_id", workerID))

	go func() {
		for {
			select {
			case <-ctx.Done():
				q.logger.Info("Worker stopping", zap.Int("worker_id", workerID))
				return
			case msg, ok := <-msgs:
				if !ok {
					q.logger.Warn("Message channel closed", zap.Int("worker_id", workerID))
					return
				}

				q.processMessage(ctx, msg, workerID)
			}
		}
	}()

	return nil
}

// processMessage handles individual job processing
func (q *QueueService) processMessage(ctx context.Context, msg amqp.Delivery, workerID int) {
	var job models.ProcessingJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		q.logger.Error("Failed to unmarshal job",
			zap.Error(err),
			zap.Int("worker_id", workerID))
		msg.Nack(false, false) // Don't requeue malformed messages
		return
	}

	q.logger.Info("Processing job",
		zap.String("job_id", job.ID),
		zap.Int("worker_id", workerID))

	// Update job status to processing
	job.Status = models.StatusProcessing

	// Process the job
	result, err := q.processJob(ctx, &job)
	if err != nil {
		job.Status = models.StatusFailed
		job.Error = err.Error()
		q.logger.Error("Job processing failed",
			zap.String("job_id", job.ID),
			zap.Error(err))
	} else {
		job.Status = models.StatusCompleted
		job.Result = result
		q.logger.Info("Job completed successfully",
			zap.String("job_id", job.ID))
	}

	// Acknowledge the message
	if err := msg.Ack(false); err != nil {
		q.logger.Error("Failed to ack message",
			zap.String("job_id", job.ID),
			zap.Error(err))
	}

	// Store job result (you might want to implement a job storage service)
	q.storeJobResult(&job)
}

// processJob performs the actual image processing
func (q *QueueService) processJob(ctx context.Context, job *models.ProcessingJob) (*models.ProcessedImage, error) {
	// For simplicity, this assumes we have the image data
	// In a real implementation, you'd fetch the image from the URL

	// Generate cache key
	cacheKey := q.storage.GenerateCacheKey(job.ImageURL, &job.Request)

	// Check cache first
	cachedData, err := q.storage.GetFromCache(ctx, cacheKey)
	if err == nil && cachedData != nil {
		q.logger.Info("Cache hit", zap.String("job_id", job.ID))
		// Return cached result (you'd need to deserialize the cached data)
	}

	// Process image (this is simplified - you'd need to fetch and process the actual image)
	result := &models.ProcessedImage{
		ID:          job.ID,
		OriginalURL: job.ImageURL,
		ProcessedAt: time.Now(),
	}

	// Cache the result
	resultBytes, _ := json.Marshal(result)
	q.storage.SetCache(ctx, cacheKey, resultBytes)

	return result, nil
}

// storeJobResult stores the job result (implement based on your storage needs)
func (q *QueueService) storeJobResult(job *models.ProcessingJob) {
	// This could store to database, cache, or file system
	// For now, we'll just log it
	q.logger.Info("Job result stored",
		zap.String("job_id", job.ID),
		zap.String("status", job.Status))
}

// GetQueueStats returns queue statistics
func (q *QueueService) GetQueueStats() (map[string]interface{}, error) {
	queueInfo, err := q.channel.QueueInspect(q.queueName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect queue: %w", err)
	}

	stats := map[string]interface{}{
		"messages":  queueInfo.Messages,
		"consumers": queueInfo.Consumers,
		"name":      queueInfo.Name,
	}

	return stats, nil
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

// HealthCheck checks if RabbitMQ is available
func (q *QueueService) HealthCheck() string {
	if q.conn == nil || q.conn.IsClosed() {
		return "unhealthy: connection closed"
	}

	if q.channel == nil {
		return "unhealthy: channel not available"
	}

	return "healthy"
}
