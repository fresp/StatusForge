package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/fresp/Statora/internal/models"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	UpdateProfile(ctx context.Context, id string, username string, passwordHash *string) error
	BeginMFAEnrollment(ctx context.Context, id string, secretEnc string, recoveryHashes []string) error
	EnableMFA(ctx context.Context, id string) error
	DisableMFA(ctx context.Context, id string) error
	ReplaceRecoveryCodes(ctx context.Context, id string, hashes []string) error
}

type MongoUserRepository struct {
	db *mongo.Database
}

func NewMongoUserRepository(db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{db: db}
}

func (r *MongoUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.Collection("users").FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.db.Collection("users").FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *MongoUserRepository) UpdateProfile(ctx context.Context, id string, username string, passwordHash *string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{"username": username}
	if passwordHash != nil {
		update["passwordHash"] = *passwordHash
	}

	_, err = r.db.Collection("users").UpdateByID(ctx, objectID, bson.M{"$set": update})
	return err
}

func (r *MongoUserRepository) BeginMFAEnrollment(ctx context.Context, id string, secretEnc string, recoveryHashes []string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.db.Collection("users").UpdateByID(ctx, objectID, bson.M{"$set": bson.M{
		"mfaSecretEnc":         secretEnc,
		"mfaRecoveryCodesHash": recoveryHashes,
		"mfaEnabled":           false,
	}})
	return err
}

func (r *MongoUserRepository) EnableMFA(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.db.Collection("users").UpdateByID(ctx, objectID, bson.M{"$set": bson.M{"mfaEnabled": true}})
	return err
}

func (r *MongoUserRepository) DisableMFA(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.db.Collection("users").UpdateByID(ctx, objectID, bson.M{"$set": bson.M{
		"mfaEnabled":           false,
		"mfaSecretEnc":         "",
		"mfaRecoveryCodesHash": []string{},
	}})
	return err
}

func (r *MongoUserRepository) ReplaceRecoveryCodes(ctx context.Context, id string, hashes []string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.db.Collection("users").UpdateByID(ctx, objectID, bson.M{"$set": bson.M{"mfaRecoveryCodesHash": hashes}})
	return err
}
