package serverKafka

import (
	ka "github.com/nvj9singhnavjot/media-docker/kafka"
)

// Global KafkaProducer variable
var KafkaProducer *ka.KafkaProducerManager // Kafka producer manager for sending messages
