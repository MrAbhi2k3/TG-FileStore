package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRecord struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    int64              `bson:"user_id" json:"user_id"`
	Username  string             `bson:"username" json:"username"`
	FirstName string             `bson:"first_name" json:"first_name"`
	LastName  string             `bson:"last_name" json:"last_name"`
	JoinedAt  time.Time          `bson:"joined_at" json:"joined_at"`
	LastSeen  time.Time          `bson:"last_seen" json:"last_seen"`
	IsBlocked bool               `bson:"is_blocked" json:"is_blocked"`
}
