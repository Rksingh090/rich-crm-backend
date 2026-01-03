package group

import (
	"context"
	"go-crm/internal/common/models"
	"go-crm/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type GroupRepository interface {
	Create(ctx context.Context, group *Group) error
	FindAll(ctx context.Context) ([]Group, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*Group, error)
	Update(ctx context.Context, id primitive.ObjectID, group *Group) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	AddMember(ctx context.Context, groupID, userID primitive.ObjectID) error
	RemoveMember(ctx context.Context, groupID, userID primitive.ObjectID) error
	FindByMember(ctx context.Context, userID primitive.ObjectID) ([]Group, error)
}

type GroupRepositoryImpl struct {
	collection *mongo.Collection
}

func NewGroupRepository(db *database.MongodbDB) GroupRepository {
	return &GroupRepositoryImpl{
		collection: db.DB.Collection("groups"),
	}
}

func (r *GroupRepositoryImpl) Create(ctx context.Context, group *Group) error {
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	if group.Members == nil {
		group.Members = []primitive.ObjectID{}
	}
	if group.Permissions == nil {
		group.Permissions = make(map[string]map[string]models.ActionPermission)
	}

	result, err := r.collection.InsertOne(ctx, group)
	if err != nil {
		return err
	}

	group.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *GroupRepositoryImpl) FindAll(ctx context.Context) ([]Group, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

func (r *GroupRepositoryImpl) FindByID(ctx context.Context, id primitive.ObjectID) (*Group, error) {
	var group Group
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepositoryImpl) Update(ctx context.Context, id primitive.ObjectID, group *Group) error {
	group.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"name":        group.Name,
			"description": group.Description,
			"permissions": group.Permissions,
			"members":     group.Members,
			"updated_at":  group.UpdatedAt,
		},
	}

	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *GroupRepositoryImpl) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id, "is_system": false})
	return err
}

func (r *GroupRepositoryImpl) AddMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	update := bson.M{
		"$addToSet": bson.M{"members": userID},
		"$set":      bson.M{"updated_at": time.Now()},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": groupID}, update)
	return err
}

func (r *GroupRepositoryImpl) RemoveMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	update := bson.M{
		"$pull": bson.M{"members": userID},
		"$set":  bson.M{"updated_at": time.Now()},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": groupID}, update)
	return err
}

func (r *GroupRepositoryImpl) FindByMember(ctx context.Context, userID primitive.ObjectID) ([]Group, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"members": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}
