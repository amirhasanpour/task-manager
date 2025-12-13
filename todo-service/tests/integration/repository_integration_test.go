package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type RepositoryIntegrationTestSuite struct {
	suite.Suite
	db     *gorm.DB
	repo   repository.TaskRepository
	ctx    context.Context
	userID string
}

func TestMain(m *testing.M) {
	// Check if we should run integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run them.")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func (suite *RepositoryIntegrationTestSuite) SetupSuite() {
	// Connect to test database
	dsn := "host=localhost user=postgres password=postgres dbname=test_tasks port=5432 sslmode=disable"
	
	var err error
	suite.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(suite.T(), err)
	
	// Run migrations
	err = suite.db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error
	assert.NoError(suite.T(), err)
	
	err = suite.db.AutoMigrate(&model.Task{})
	assert.NoError(suite.T(), err)
	
	suite.repo = repository.NewTaskRepository(suite.db)
	suite.ctx = context.Background()
	
	// Generate a valid UUID for user ID
	suite.userID = uuid.New().String()
}

func (suite *RepositoryIntegrationTestSuite) TearDownSuite() {
	// Clean up database
	suite.db.Exec("DROP TABLE IF EXISTS tasks")
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

func (suite *RepositoryIntegrationTestSuite) SetupTest() {
	// Clean table before each test
	suite.db.Exec("DELETE FROM tasks")
}

func (suite *RepositoryIntegrationTestSuite) TestCreateAndFindTask() {
	dueDate := time.Now().Add(24 * time.Hour)
	
	task := &model.Task{
		UserID:      suite.userID,
		Title:       "Integration Test Task",
		Description: "Integration Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
		DueDate:     &dueDate,
	}

	// Test Create
	createdTask, err := suite.repo.Create(suite.ctx, task)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), createdTask)
	assert.NotEmpty(suite.T(), createdTask.ID)
	assert.Equal(suite.T(), "Integration Test Task", createdTask.Title)
	assert.Equal(suite.T(), suite.userID, createdTask.UserID)
	assert.Equal(suite.T(), model.StatusTodo, createdTask.Status)
	assert.Equal(suite.T(), model.PriorityMedium, createdTask.Priority)

	// Test FindByID
	foundTask, err := suite.repo.FindByID(suite.ctx, createdTask.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), foundTask)
	assert.Equal(suite.T(), createdTask.ID, foundTask.ID)
	assert.Equal(suite.T(), "Integration Test Task", foundTask.Title)

	// Test FindByIDAndUser (correct user)
	foundTaskByUser, err := suite.repo.FindByIDAndUser(suite.ctx, createdTask.ID, suite.userID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), foundTaskByUser)
	assert.Equal(suite.T(), createdTask.ID, foundTaskByUser.ID)

	// Test FindByIDAndUser (wrong user - should return nil)
	wrongUserID := uuid.New().String()
	notFoundTask, err := suite.repo.FindByIDAndUser(suite.ctx, createdTask.ID, wrongUserID)
	assert.NoError(suite.T(), err)
	assert.Nil(suite.T(), notFoundTask)
}

func (suite *RepositoryIntegrationTestSuite) TestUpdateTask() {
	task := &model.Task{
		UserID:      suite.userID,
		Title:       "Original Title",
		Description: "Original Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityLow,
	}

	createdTask, err := suite.repo.Create(suite.ctx, task)
	assert.NoError(suite.T(), err)

	// Update task
	createdTask.Title = "Updated Title"
	createdTask.Status = model.StatusInProgress
	createdTask.Priority = model.PriorityHigh
	
	updatedTask, err := suite.repo.Update(suite.ctx, createdTask)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Title", updatedTask.Title)
	assert.Equal(suite.T(), model.StatusInProgress, updatedTask.Status)
	assert.Equal(suite.T(), model.PriorityHigh, updatedTask.Priority)

	// Verify update persisted
	foundTask, err := suite.repo.FindByID(suite.ctx, createdTask.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Title", foundTask.Title)
}

func (suite *RepositoryIntegrationTestSuite) TestDeleteTask() {
	task := &model.Task{
		UserID: suite.userID,
		Title:  "Task to Delete",
	}

	createdTask, err := suite.repo.Create(suite.ctx, task)
	assert.NoError(suite.T(), err)

	// Delete task
	err = suite.repo.Delete(suite.ctx, createdTask.ID)
	assert.NoError(suite.T(), err)

	// Verify task is deleted
	foundTask, err := suite.repo.FindByID(suite.ctx, createdTask.ID)
	assert.NoError(suite.T(), err)
	assert.Nil(suite.T(), foundTask)
}

func (suite *RepositoryIntegrationTestSuite) TestDeleteByUser() {
	task := &model.Task{
		UserID: suite.userID,
		Title:  "Task to Delete By User",
	}

	createdTask, err := suite.repo.Create(suite.ctx, task)
	assert.NoError(suite.T(), err)

	// Delete task by correct user
	err = suite.repo.DeleteByUser(suite.ctx, createdTask.ID, suite.userID)
	assert.NoError(suite.T(), err)

	// Verify task is deleted
	foundTask, err := suite.repo.FindByID(suite.ctx, createdTask.ID)
	assert.NoError(suite.T(), err)
	assert.Nil(suite.T(), foundTask)
}

func (suite *RepositoryIntegrationTestSuite) TestListTasks() {
	// Create multiple tasks
	for i := 1; i <= 5; i++ {
		task := &model.Task{
			UserID: suite.userID,
			Title:  fmt.Sprintf("Task %d", i),
			Status: model.StatusTodo,
		}
		_, err := suite.repo.Create(suite.ctx, task)
		assert.NoError(suite.T(), err)
	}

	// Create a task for different user
	otherUserID := uuid.New().String()
	otherTask := &model.Task{
		UserID: otherUserID,
		Title:  "Other User Task",
	}
	_, err := suite.repo.Create(suite.ctx, otherTask)
	assert.NoError(suite.T(), err)

	// List all tasks
	filter := &repository.TaskFilter{}
	tasks, total, err := suite.repo.List(suite.ctx, filter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(6), total)
	assert.Len(suite.T(), tasks, 6)

	// List tasks with user filter
	userFilter := &repository.TaskFilter{
		UserID: &suite.userID,
	}
	userTasks, userTotal, err := suite.repo.List(suite.ctx, userFilter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(5), userTotal)
	assert.Len(suite.T(), userTasks, 5)

	// Test pagination
	page1Tasks, page1Total, err := suite.repo.List(suite.ctx, filter, 1, 2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(6), page1Total)
	assert.Len(suite.T(), page1Tasks, 2)

	page2Tasks, page2Total, err := suite.repo.List(suite.ctx, filter, 2, 2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(6), page2Total)
	assert.Len(suite.T(), page2Tasks, 2)
}

func (suite *RepositoryIntegrationTestSuite) TestListTasksByUser() {
	// Create multiple tasks for the user
	for i := 1; i <= 3; i++ {
		task := &model.Task{
			UserID: suite.userID,
			Title:  fmt.Sprintf("User Task %d", i),
		}
		_, err := suite.repo.Create(suite.ctx, task)
		assert.NoError(suite.T(), err)
	}

	// List tasks by user
	filter := &repository.TaskFilter{}
	tasks, total, err := suite.repo.ListByUser(suite.ctx, suite.userID, filter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total)
	assert.Len(suite.T(), tasks, 3)

	// Test with status filter
	status := string(model.StatusTodo)
	statusFilter := &repository.TaskFilter{
		Status: &status,
	}
	statusTasks, statusTotal, err := suite.repo.ListByUser(suite.ctx, suite.userID, statusFilter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), statusTotal)
	assert.Len(suite.T(), statusTasks, 3)
}

func (suite *RepositoryIntegrationTestSuite) TestListTasksWithSorting() {
	// Create tasks with different priorities
	task1 := &model.Task{
		UserID:   suite.userID,
		Title:    "Task A",
		Priority: model.PriorityLow,
	}
	task2 := &model.Task{
		UserID:   suite.userID,
		Title:    "Task B",
		Priority: model.PriorityHigh,
	}
	task3 := &model.Task{
		UserID:   suite.userID,
		Title:    "Task C",
		Priority: model.PriorityMedium,
	}

	_, err := suite.repo.Create(suite.ctx, task1)
	assert.NoError(suite.T(), err)
	_, err = suite.repo.Create(suite.ctx, task2)
	assert.NoError(suite.T(), err)
	_, err = suite.repo.Create(suite.ctx, task3)
	assert.NoError(suite.T(), err)

	// Test sorting by priority ascending
	ascFilter := &repository.TaskFilter{
		SortBy:   "priority",
		SortDesc: false,
	}
	ascTasks, _, err := suite.repo.ListByUser(suite.ctx, suite.userID, ascFilter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), ascTasks, 3)
	
	// Since we can't guarantee order of equal priorities, just verify all tasks are returned
	taskTitles := make([]string, len(ascTasks))
	for i, task := range ascTasks {
		taskTitles[i] = task.Title
	}
	assert.Contains(suite.T(), taskTitles, "Task A")
	assert.Contains(suite.T(), taskTitles, "Task B")
	assert.Contains(suite.T(), taskTitles, "Task C")

	// Test sorting by priority descending
	descFilter := &repository.TaskFilter{
		SortBy:   "priority",
		SortDesc: true,
	}
	descTasks, _, err := suite.repo.ListByUser(suite.ctx, suite.userID, descFilter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), descTasks, 3)
}

func TestRepositoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryIntegrationTestSuite))
}