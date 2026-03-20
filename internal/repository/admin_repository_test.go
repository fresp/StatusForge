package repository

import "testing"

func TestMongoUserRepositoryImplementsUserRepository(t *testing.T) {
	var _ UserRepository = (*MongoUserRepository)(nil)
}
