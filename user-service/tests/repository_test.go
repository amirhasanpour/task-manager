package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/amirhasanpour/task-manager/user-service/internal/model"
	"github.com/amirhasanpour/task-manager/user-service/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type UserRepositoryTestSuite struct {
	suite.Suite
	db     *gorm.DB
	repo   repository.UserRepository
	ctx    context.Context
}

func (suite *UserRepositoryTestSuite) SetupTest() {
	// Create in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Create UUID extension (not needed for SQLite)
	// db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	// Auto migrate
	err = db.AutoMigrate(&model.User{})
	assert.NoError(suite.T(), err)

	suite.db = db
	suite.repo = repository.NewUserRepository(db)
	suite.ctx = context.Background()
}

func (suite *UserRepositoryTestSuite) TearDownTest() {
	// Clean up
	sqlDB, err := suite.db.DB()
	assert.NoError(suite.T(), err)
	sqlDB.Close()
}

func (suite *UserRepositoryTestSuite) TestCreateUser() {
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		FullName: "Test User",
	}

	createdUser, err := suite.repo.Create(suite.ctx, user)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), createdUser)
	assert.NotEmpty(suite.T(), createdUser.ID)
	assert.Equal(suite.T(), "testuser", createdUser.Username)
	assert.Equal(suite.T(), "test@example.com", createdUser.Email)
}

func (suite *UserRepositoryTestSuite) TestFindByID() {
	// First create a user
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		FullName: "Test User",
	}

	createdUser, err := suite.repo.Create(suite.ctx, user)
	assert.NoError(suite.T(), err)

	// Then find by ID
	foundUser, err := suite.repo.FindByID(suite.ctx, createdUser.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), foundUser)
	assert.Equal(suite.T(), createdUser.ID, foundUser.ID)
	assert.Equal(suite.T(), "testuser", foundUser.Username)
}

func (suite *UserRepositoryTestSuite) TestFindByEmail() {
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		FullName: "Test User",
	}

	_, err := suite.repo.Create(suite.ctx, user)
	assert.NoError(suite.T(), err)

	foundUser, err := suite.repo.FindByEmail(suite.ctx, "test@example.com")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), foundUser)
	assert.Equal(suite.T(), "test@example.com", foundUser.Email)
}

func (suite *UserRepositoryTestSuite) TestUpdateUser() {
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		FullName: "Test User",
	}

	createdUser, err := suite.repo.Create(suite.ctx, user)
	assert.NoError(suite.T(), err)

	// Update user
	createdUser.FullName = "Updated Name"
	updatedUser, err := suite.repo.Update(suite.ctx, createdUser)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Name", updatedUser.FullName)
}

func (suite *UserRepositoryTestSuite) TestDeleteUser() {
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		FullName: "Test User",
	}

	createdUser, err := suite.repo.Create(suite.ctx, user)
	assert.NoError(suite.T(), err)

	// Delete user
	err = suite.repo.Delete(suite.ctx, createdUser.ID)
	assert.NoError(suite.T(), err)

	// Verify user is deleted
	foundUser, err := suite.repo.FindByID(suite.ctx, createdUser.ID)
	assert.NoError(suite.T(), err)
	assert.Nil(suite.T(), foundUser)
}

func (suite *UserRepositoryTestSuite) TestListUsers() {
	// Create multiple users
	for i := 1; i <= 5; i++ {
		user := &model.User{
			Username:  "testuser" + string(rune('0'+i)),
			Email:     "test" + string(rune('0'+i)) + "@example.com",
			Password:  "hashedpassword",
			FullName:  "Test User " + string(rune('0'+i)),
		}
		_, err := suite.repo.Create(suite.ctx, user)
		assert.NoError(suite.T(), err)
	}

	// List users with pagination
	users, total, err := suite.repo.List(suite.ctx, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(5), total)
	assert.Len(suite.T(), users, 5)
}

func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}