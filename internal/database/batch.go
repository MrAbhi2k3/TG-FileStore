package database

import (
	"context"
	"time"

	"github.com/MrAbhi2k3/TG-FileStore/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (m *MongoDB) StartBatchSession(ctx context.Context, userID int64) error {
	coll := m.Database.Collection("batch_sessions")
	now := time.Now().UTC()
	opts := options.Update().SetUpsert(true)
	_, err := coll.UpdateOne(ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"created_at": now, "status_msg_id": 0, "status_chat_id": 0}},
		opts,
	)
	return err
}

func (m *MongoDB) GetBatchSession(ctx context.Context, userID int64) (*models.BatchSession, error) {
	coll := m.Database.Collection("batch_sessions")
	var session models.BatchSession
	err := coll.FindOne(ctx, bson.M{"user_id": userID}).Decode(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (m *MongoDB) SetBatchStatusMsg(ctx context.Context, userID int64, chatID int64, messageID int) error {
	coll := m.Database.Collection("batch_sessions")
	_, err := coll.UpdateOne(ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"status_chat_id": chatID, "status_msg_id": messageID}},
	)
	return err
}

func (m *MongoDB) IsInBatchSession(ctx context.Context, userID int64) bool {
	coll := m.Database.Collection("batch_sessions")
	count, err := coll.CountDocuments(ctx, bson.M{"user_id": userID})
	return err == nil && count > 0
}

func (m *MongoDB) AddToBatchQueue(ctx context.Context, item *models.BatchQueueItem) error {
	coll := m.Database.Collection("batch_queue")
	item.CreatedAt = time.Now().UTC()
	_, err := coll.InsertOne(ctx, item)
	return err
}

func (m *MongoDB) GetBatchQueue(ctx context.Context, userID int64) ([]models.BatchQueueItem, error) {
	coll := m.Database.Collection("batch_queue")
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := coll.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []models.BatchQueueItem
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *MongoDB) GetBatchQueueCount(ctx context.Context, userID int64) int64 {
	coll := m.Database.Collection("batch_queue")
	count, _ := coll.CountDocuments(ctx, bson.M{"user_id": userID})
	return count
}

func (m *MongoDB) ClearBatch(ctx context.Context, userID int64) error {
	_, _ = m.Database.Collection("batch_sessions").DeleteOne(ctx, bson.M{"user_id": userID})
	_, err := m.Database.Collection("batch_queue").DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}
