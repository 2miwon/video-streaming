package main

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
    ID       string `bson:"_id,omitempty"`
	Email	 string `bson:"email"`
    Username string `bson:"username"`
    Password string `bson:"password"`
	Created time.Time `bson:"created"`
	Token string `bson:"token"`
	History []VideoHistory `bson:"history"`
	BookMark []string `bson:"bookmark"`
}

type Video struct {
	ID       string `bson:"_id,omitempty"`
	Title	 string `bson:"title"`
	Content  string `bson:"content"`
	URL 	 string `bson:"url"`
	// ThumbnailURL *string `bson:"thumbnail_url"`
	AuthorID string `bson:"author_id"`
	Created time.Time `bson:"created"`
	Deleted *time.Time `bson:"deleted"`
}

type VideoHistory struct {
    VideoID string    `json:"video_id"`
    Date    time.Time `json:"date"`
}

func connectDB(uri string) (*mongo.Client, context.Context, error) {
	// context는 일정 시간이 지나면 자동으로 취소
	// 단발성 연결 시에
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()
	ctx := context.Background()
	
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)
	// client, err := mongo.Connect(context.TODO(), opts)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, ctx, err
	}
	return client, ctx, nil
}

func createUser(collection *mongo.Collection, ctx context.Context, json User) (*mongo.InsertOneResult, error) {
	rst, err := collection.InsertOne(ctx, json)
	if err != nil {
		return nil, err
	}
	return rst, nil
}

func checkDocumentExists(collection *mongo.Collection, ctx context.Context, filter bson.M, message string) error {
	num, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return err
	}

	if num == 0 {
		return fmt.Errorf(message)
	}

	return nil
}

func checkDocumentNotExists(collection *mongo.Collection, ctx context.Context, filter bson.M, message string) error {
	num, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return err
	}

	if num != 0 {
		return fmt.Errorf(message)
	}

	return nil
}