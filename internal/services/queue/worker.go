package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/phambaophuc/image-resize/internal/models"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

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

// storeJobResult stores the job result (implement based on your storage needs)
func (q *QueueService) storeJobResult(job *models.ProcessingJob) {
	// This could store to database, cache, or file system
	// For now, we'll just log it
	q.logger.Info("Job result stored",
		zap.String("job_id", job.ID),
		zap.String("status", job.Status))
}
