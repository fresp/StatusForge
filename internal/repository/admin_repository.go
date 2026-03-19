package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"status-platform/internal/models"
)

type AdminRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.Admin, error)
}

type MongoAdminRepository struct {
	db *mongo.Database
}

func NewMongoAdminRepository(db *mongo.Database) *MongoAdminRepository {
	return &MongoAdminRepository{db: db}
}

func (r *MongoAdminRepository) FindByEmail(ctx context.Context, email string) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.Collection("admins").FindOne(ctx, bson.M{"email": email}).Decode(&admin)
	if err != nil {
		return nil, err
	}
	return &admin, nil
}
