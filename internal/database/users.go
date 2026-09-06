package database

import (
	"context"
	"time"

	"github.com/MrAbhi2k3/TG-FileStore/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (m *MongoDB) TrackUser(ctx context.Context, u *models.UserRecord) error {
	coll := m.Database.Collection("users")
	now := time.Now().UTC()

	filter := bson.M{"user_id": u.UserID}
	update := bson.M{
		"$set": bson.M{
			"username":   u.Username,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
			"last_seen":  now,
		},
		"$setOnInsert": bson.M{
			"joined_at":  now,
			"is_blocked": false,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := coll.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoDB) GetTotalUsers(ctx context.Context) (int64, error) {
	coll := m.Database.Collection("users")
	return coll.CountDocuments(ctx, bson.M{})
}

func (m *MongoDB) GetAllUserIDs(ctx context.Context) ([]int64, error) {
	coll := m.Database.Collection("users")
	opts := options.Find().SetProjection(bson.M{"user_id": 1, "_id": 0})
	cursor, err := coll.Find(ctx, bson.M{"is_blocked": bson.M{"$ne": true}}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	type userProj struct {
		UserID int64 `bson:"user_id"`
	}
	var results []userProj
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	ids := make([]int64, len(results))
	for i, r := range results {
		ids[i] = r.UserID
	}
	return ids, nil
}

func (m *MongoDB) MarkUserBlocked(ctx context.Context, userID int64) error {
	coll := m.Database.Collection("users")
	_, err := coll.UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$set": bson.M{"is_blocked": true}})
	return err
}
