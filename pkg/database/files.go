package database

import (
	"context"
	"errors"
	"time"

	"github.com/MrAbhi2k3/TG-FileStore/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FileStats struct {
	TotalFiles     int64 `bson:"total_files"`
	TotalSize      int64 `bson:"total_size"`
	TotalDownloads int64 `bson:"total_downloads"`
}

func (m *MongoDB) ensureIndexes(ctx context.Context) error {
	filesColl := m.Database.Collection("files")
	usersColl := m.Database.Collection("users")

	fileIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "hash", Value: 1}},
			Options: options.Index().SetName("idx_hash"),
		},
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_token_unique"),
		},
		{
			Keys:    bson.D{{Key: "message_id", Value: 1}},
			Options: options.Index().SetName("idx_message_id"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
	}
	_, _ = filesColl.Indexes().CreateMany(ctx, fileIndexes)

	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_user_id_unique"),
		},
		{
			Keys:    bson.D{{Key: "joined_at", Value: -1}},
			Options: options.Index().SetName("idx_joined_at"),
		},
	}
	_, _ = usersColl.Indexes().CreateMany(ctx, userIndexes)

	return nil
}

func (m *MongoDB) FindFileByToken(ctx context.Context, token string) (*models.FileRecord, error) {
	coll := m.Database.Collection("files")
	var record models.FileRecord
	err := coll.FindOne(ctx, bson.M{"token": token}).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (m *MongoDB) FindFileByHash(ctx context.Context, hash string) (*models.FileRecord, error) {
	coll := m.Database.Collection("files")
	var record models.FileRecord
	err := coll.FindOne(ctx, bson.M{"hash": hash}).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (m *MongoDB) SaveFile(ctx context.Context, record *models.FileRecord) error {
	coll := m.Database.Collection("files")
	now := time.Now().UTC()
	record.CreatedAt = now
	record.UpdatedAt = now

	res, err := coll.InsertOne(ctx, record)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		record.ID = oid
	}
	return nil
}

func (m *MongoDB) IncrementDownloads(ctx context.Context, token string) error {
	coll := m.Database.Collection("files")
	_, err := coll.UpdateOne(ctx,
		bson.M{"token": token},
		bson.M{
			"$inc": bson.M{"downloads": 1},
			"$set": bson.M{"updated_at": time.Now().UTC()},
		},
	)
	return err
}

func (m *MongoDB) DeleteFileByHash(ctx context.Context, hash string) (*models.FileRecord, error) {
	coll := m.Database.Collection("files")
	var record models.FileRecord
	err := coll.FindOneAndDelete(ctx, bson.M{"hash": hash}).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (m *MongoDB) DeleteFileByToken(ctx context.Context, token string) (*models.FileRecord, error) {
	coll := m.Database.Collection("files")
	var record models.FileRecord
	err := coll.FindOneAndDelete(ctx, bson.M{"token": token}).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (m *MongoDB) GetRecentFiles(ctx context.Context, limit int64) ([]*models.FileRecord, error) {
	coll := m.Database.Collection("files")
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cursor, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var files []*models.FileRecord
	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (m *MongoDB) GetFileStats(ctx context.Context) (*FileStats, error) {
	coll := m.Database.Collection("files")

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total_files", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "total_size", Value: bson.D{{Key: "$sum", Value: "$file_size"}}},
			{Key: "total_downloads", Value: bson.D{{Key: "$sum", Value: "$downloads"}}},
		}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stats []FileStats
	if err := cursor.All(ctx, &stats); err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return &FileStats{}, nil
	}
	return &stats[0], nil
}
