package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FileType string

const (
	FileTypeDocument  FileType = "document"
	FileTypeVideo     FileType = "video"
	FileTypeAudio     FileType = "audio"
	FileTypePhoto     FileType = "photo"
	FileTypeVoice     FileType = "voice"
	FileTypeAnimation FileType = "animation"
	FileTypeSticker   FileType = "sticker"
	FileTypeUnknown   FileType = "unknown"
)

type FileRecord struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Hash         string             `bson:"hash" json:"hash"`
	Token        string             `bson:"token" json:"token"`
	MessageID    int                `bson:"message_id" json:"message_id"`
	ChannelID    int64              `bson:"channel_id" json:"channel_id"`
	FileName     string             `bson:"file_name" json:"file_name"`
	FileSize     int64              `bson:"file_size" json:"file_size"`
	MimeType     string             `bson:"mime_type" json:"mime_type"`
	FileType     FileType           `bson:"file_type" json:"file_type"`
	FileID       string             `bson:"file_id" json:"file_id"`
	UniqueFileID string             `bson:"file_unique_id" json:"file_unique_id"`
	UploadedBy   int64              `bson:"uploaded_by" json:"uploaded_by"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	Downloads    int64              `bson:"downloads" json:"downloads"`
	IsBatch      bool               `bson:"is_batch,omitempty" json:"is_batch"`
	MessageIDs   []int              `bson:"message_ids,omitempty" json:"message_ids"`
}
