package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

var (
	mongoInstance *MongoDB
	mongoOnce     sync.Once
	mongoErr      error
)

func GetMongoDB(ctx context.Context, uri string, dbName string) (*MongoDB, error) {
	mongoOnce.Do(func() {
		clientOptions := options.Client().
			ApplyURI(uri).
			SetMaxPoolSize(50).
			SetMinPoolSize(1).
			SetMaxConnIdleTime(60 * time.Second).
			SetConnectTimeout(10 * time.Second)

		connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		client, err := mongo.Connect(connectCtx, clientOptions)
		if err != nil {
			mongoErr = fmt.Errorf("failed to connect to mongodb: %w", err)
			return
		}

		if err := client.Ping(connectCtx, nil); err != nil {
			mongoErr = fmt.Errorf("failed to ping mongodb: %w", err)
			return
		}

		db := client.Database(dbName)
		mongoInstance = &MongoDB{
			Client:   client,
			Database: db,
		}

		initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer initCancel()
		_ = mongoInstance.ensureIndexes(initCtx)
	})

	if mongoErr != nil {
		return nil, mongoErr
	}
	return mongoInstance, nil
}
