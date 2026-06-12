package constants

// LavinMQ / RabbitMQ topic exchange defaults (CloudAMQP recommends amq.topic).
const (
	LavinMQExchangeDefault = "amq.topic"
	LavinMQQueuePrefix     = "mycourse"
)

// Topic routing keys published on the configured topic exchange.
const (
	TopicHealthPing           = "mycourse.health.ping"
	TopicMediaUploadCompleted = "mycourse.media.upload.completed"
	TopicCoursePublished      = "mycourse.course.published"
)
