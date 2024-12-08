package queue

import (
	"context"
	"errors"
	"product-management-system/db"
	"strings"

	"github.com/streadway/amqp"
)

// StartConsumer listens for messages on a RabbitMQ queue
func StartConsumer(queueName string) {
	msgs, err := Channel.Consume(
		queueName, // Queue name
		"",        // Consumer tag
		false,     // Auto-ack (manual ack for retry logic)
		false,     // Exclusive
		false,     // No-local
		false,     // No-wait
		nil,       // Args
	)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to register a consumer")
	}

	Logger.WithField("queue", queueName).Info("Waiting for messages on queue")

	for msg := range msgs {
		Logger.WithField("message", string(msg.Body)).Info("Received message")

		// Process the image
		err := processImage(string(msg.Body))
		if err != nil {
			Logger.WithError(err).WithField("message", string(msg.Body)).Error("Processing failed, sending to retry queue")

			// Publish to the retry queue
			err = Channel.Publish(
				"",                       // Default exchange
				"image_processing_retry", // Retry queue
				false,                    // Mandatory
				false,                    // Immediate
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        msg.Body,
				},
			)
			if err != nil {
				Logger.WithError(err).Error("Failed to send message to retry queue")
			}

			// Reject the message without requeueing (sends to DLQ)
			msg.Nack(false, false) // Multiple=false, Requeue=false
			continue
		}

		// Acknowledge successful processing
		msg.Ack(false)
		Logger.WithField("message", string(msg.Body)).Info("Message processed successfully")
	}
}

// processImage simulates image processing (compression) and updates the database
func processImage(imageURL string) error {
	if strings.Contains(imageURL, "fail") {
		return errors.New("simulated processing failure")
	}
	compressedImageURL := strings.Replace(imageURL, ".", "-compressed.", 1)
	return updateCompressedImagesInDB(compressedImageURL)
}

// updateCompressedImagesInDB updates the compressed images in the database
func updateCompressedImagesInDB(compressedImageURL string) error {
	ctx := context.Background()

	// Replace `1` with the actual product ID in production
	_, err := db.DB.Exec(ctx, `
		UPDATE products
		SET compressed_product_images = array_append(compressed_product_images, $1)
		WHERE id = $2
	`, compressedImageURL, 1)
	return err
}
