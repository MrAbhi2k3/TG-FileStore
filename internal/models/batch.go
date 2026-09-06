package models

import "time"

type BatchQueueItem struct {
	UserID    int64     `bson:"user_id"`
	MessageID int       `bson:"message_id"`
	FileName  string    `bson:"file_name"`
	FileSize  int64     `bson:"file_size"`
	CreatedAt time.Time `bson:"created_at"`
}

type BatchSession struct {
	UserID    int64     `bson:"user_id"`
	CreatedAt time.Time `bson:"created_at"`
}
