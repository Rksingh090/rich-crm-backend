package repository

import (
	"context"
	"go-crm/internal/database"
	"go-crm/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type GroupRepository struct {
	collection *mongo.Collection
}

func NewGroupRepository(db *database.MongodbDB) *GroupRepository {
	return &GroupRepository{
		collection: db.DB.Collection("groups"),
	}
}

func (r *GroupRepository) Create(ctx context.Context, group *models.Group) error {
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	if group.Members == nil {
		group.Members = []primitive.ObjectID{}
	}
	if group.ModulePermissions == nil {
		group.ModulePermissions = make(map[string]models.ModulePermission)
	}

	result, err := r.collection.InsertOne(ctx, group)
	if err != nil {
		return err
	}

	group.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *GroupRepository) FindAll(ctx context.Context) ([]models.Group, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

func (r *GroupRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Group, error) {
	var group models.Group
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepository) Update(ctx context.Context, id primitive.ObjectID, group *models.Group) error {
	group.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"name":               group.Name,
			"description":        group.Description,
			"module_permissions": group.ModulePermissions,
			"members":            group.Members,
			"updated_at":         group.UpdatedAt,
		},
	}

	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *GroupRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id, "is_system": false})
	return err
}

func (r *GroupRepository) AddMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	update := bson.M{
		"$addToSet": bson.M{"members": userID},
		"$set":      bson.M{"updated_at": time.Now()},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": groupID}, update)
	return err
}

func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, userID primitive.ObjectID) error {
	update := bson.M{
		"$pull": bson.M{"members": userID},
		"$set":  bson.M{"updated_at": time.Now()},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": groupID}, update)
	return err
}

func (r *GroupRepository) FindByMember(ctx context.Context, userID primitive.ObjectID) ([]models.Group, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"members": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}
