package queue

import (
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var Conn *amqp.Connection
var Channel *amqp.Channel

// InitQueue initializes RabbitMQ connection and channel
func InitQueue() {
	var err error
	Conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		Logger.WithError(err).Fatal("Failed to connect to RabbitMQ")
	}

	Channel, err = Conn.Channel()
	if err != nil {
		Logger.WithError(err).Fatal("Failed to open a channel")
	}

	// Declare Dead-Letter Queue
	_, err = Channel.QueueDeclare(
		"image_processing_dlx", // DLX (Dead-Letter Queue)
		true,                   // Durable
		false,                  // Delete when unused
		false,                  // Exclusive
		false,                  // No-wait
		nil,                    // Arguments
	)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to declare Dead-Letter Queue")
	}

	// Declare Retry Queue with DLX binding
	_, err = Channel.QueueDeclare(
		"image_processing_retry", // Retry Queue
		true,                     // Durable
		false,                    // Delete when unused
		false,                    // Exclusive
		false,                    // No-wait
		amqp.Table{
			"x-dead-letter-exchange":    "",                 // Default exchange
			"x-dead-letter-routing-key": "image_processing", // Route back to main queue
			"x-message-ttl":             10000,              // Retry after 10 seconds
		},
	)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to declare Retry Queue")
	}

	// Declare Main Queue with DLX binding
	_, err = Channel.QueueDeclare(
		"image_processing", // Main Queue
		true,               // Durable
		false,              // Delete when unused
		false,              // Exclusive
		false,              // No-wait
		amqp.Table{
			"x-dead-letter-exchange":    "",                     // Default exchange
			"x-dead-letter-routing-key": "image_processing_dlx", // Route to DLX on failure
		},
	)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to declare Main Queue")
	}

	Logger.Info("RabbitMQ connection and channel initialized with DLQ support")
}

// Publish sends a message to the specified queue
func Publish(queueName, message string) {
	_, err := Channel.QueueDeclare(
		queueName, // Queue name
		false,     // Durable
		false,     // Delete when unused
		false,     // Exclusive
		false,     // No-wait
		nil,       // Arguments
	)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to declare a queue")
	}

	err = Channel.Publish(
		"",        // Exchange
		queueName, // Routing key
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to publish a message")
	}

	Logger.WithFields(logrus.Fields{
		"queue":   queueName,
		"message": message,
	}).Info("Message published to queue successfully")
}
