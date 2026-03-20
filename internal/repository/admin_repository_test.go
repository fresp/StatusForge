package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMongoUserRepositorySupportsMFAUpdateMethods(t *testing.T) {
	var repo UserRepository = &MongoUserRepository{}
	assert.Implements(t, (*UserRepository)(nil), repo)
}
