package queue

import "fmt"

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
