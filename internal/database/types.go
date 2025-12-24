package database

import "go.mongodb.org/mongo-driver/mongo"

type MongodbDB struct {
	DB *mongo.Database
}
