package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/phambaophuc/image-resize/internal/models"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

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
