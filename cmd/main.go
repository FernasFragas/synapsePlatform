package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"synapsePlatform/internal/ingestor"
	"syscall"

	"synapsePlatform/internal"
)

type application struct {
	db       *internal.DB
	consumer ingestor.MessagePoller
}

func main() {
	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Open database
	db, err := internal.Open("data.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Db.Close()

	// Configure Kafka consumer
	kafkaConfig := internal.StreamingConfigs{
		Brokers:  []string{"localhost:9092"},
		GroupID:  "synapse-platform-consumer",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	}

	// Create Kafka consumer
	consumer := internal.NewConsumer(kafkaConfig)

	// Subscribe to the ingestion topic
	if err := consumer.Subscribe("ingestion.raw"); err != nil {
		log.Fatalf("Failed to subscribe to topics: %v", err)
	}

	// Create message handler that logs received messages
	messageHandler := func(ctx context.Context, msg ingestor.DeviceMessage) error {
		log.Println("═══════════════════════════════════════")
		log.Printf("📨 New Message Received")
		log.Println("───────────────────────────────────────")
		log.Printf("  Device ID:  %s", msg.DeviceID)
		log.Printf("  Type:       %s", msg.Type)
		log.Printf("  Timestamp:  %s", msg.Timestamp)
		log.Printf("  Metrics:    %v", formatMetrics(msg.Metrics))
		log.Println("═══════════════════════════════════════")
		return nil
	}

	// PollMessage consuming messages
	if err := consumer.RetrieveMessage(ctx, messageHandler); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	// Log startup information
	log.Println("\n✅ Synapse Platform Started Successfully")
	log.Println("📡 Kafka Consumer: Active")
	log.Println("📥 Topic: ingestion.raw")
	log.Println("🔌 Broker: localhost:9092")
	log.Println("💾 Database: data.db")
	log.Println("\n⏳ Waiting for messages... (Press Ctrl+C to stop)")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("\n\n🛑 Shutdown signal received")
	log.Println("🔄 Closing Kafka consumer...")
	cancel()

	if err := consumer.Close(); err != nil {
		log.Printf("⚠️  Error closing consumer: %v", err)
	} else {
		log.Println("✅ Kafka consumer closed")
	}

	log.Println("✅ Shutdown complete")
}

// formatMetrics formats the metrics map for pretty printing
func formatMetrics(metrics map[string]any) string {
	if len(metrics) == 0 {
		return "{}"
	}

	result := "{\n"
	for key, value := range metrics {
		result += fmt.Sprintf("      %s: %v\n", key, value)
	}
	result += "    }"
	return result
}
