package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TaskRepositoryTestSuite struct {
	suite.Suite
	db     *gorm.DB
	repo   repository.TaskRepository
	ctx    context.Context
	userID string
}

func (suite *TaskRepositoryTestSuite) SetupTest() {
	// Create in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Auto migrate
	err = db.AutoMigrate(&model.Task{})
	assert.NoError(suite.T(), err)

	suite.db = db
	suite.repo = repository.NewTaskRepository(db)
	suite.ctx = context.Background()
	suite.userID = "test-user-id"
}

func (suite *TaskRepositoryTestSuite) TearDownTest() {
	// Clean up
	sqlDB, err := suite.db.DB()
	assert.NoError(suite.T(), err)
	sqlDB.Close()
}

func (suite *TaskRepositoryTestSuite) TestCreateTask() {
	dueDate := time.Now().Add(24 * time.Hour)
	
	task := &model.Task{
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
		DueDate:     &dueDate,
	}

	createdTask, err := suite.repo.Create(suite.ctx, task)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), createdTask)
	assert.NotEmpty(suite.T(), createdTask.ID)
	assert.Equal(suite.T(), "Test Task", createdTask.Title)
	assert.Equal(suite.T(), suite.userID, createdTask.UserID)
	assert.Equal(suite.T(), model.StatusTodo, createdTask.Status)
	assert.Equal(suite.T(), model.PriorityMedium, createdTask.Priority)
	assert.NotNil(suite.T(), createdTask.CreatedAt)
	assert.NotNil(suite.T(), createdTask.UpdatedAt)
}

func (suite *TaskRepositoryTestSuite) TestFindByID() {
	task := &model.Task{
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
	}

	createdTask, err := suite.repo.Create(suite.ctx, task)
	assert.NoError(suite.T(), err)

	// Then find by ID
	foundTask, err := suite.repo.FindByID(suite.ctx, createdTask.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), foundTask)
	assert.Equal(suite.T(), createdTask.ID, foundTask.ID)
	assert.Equal(suite.T(), "Test Task", foundTask.Title)
}

func (suite *TaskRepositoryTestSuite) TestFindByIDAndUser() {
	task := &model.Task{
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
	}

	createdTask, err := suite.repo.Create(suite.ctx, task)
	assert.NoError(suite.T(), err)

	// Find by ID and correct user
	foundTask, err := suite.repo.FindByIDAndUser(suite.ctx, createdTask.ID, suite.userID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), foundTask)
	assert.Equal(suite.T(), createdTask.ID, foundTask.ID)

	// Find by ID and wrong user - should return nil
	notFoundTask, err := suite.repo.FindByIDAndUser(suite.ctx, createdTask.ID, "wrong-user-id")
	assert.NoError(suite.T(), err)
	assert.Nil(suite.T(), notFoundTask)
}

func (suite *TaskRepositoryTestSuite) TestUpdateTask() {
	task := &model.Task{
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
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
}

func (suite *TaskRepositoryTestSuite) TestDeleteTask() {
	task := &model.Task{
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
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

func (suite *TaskRepositoryTestSuite) TestDeleteByUser() {
	task := &model.Task{
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
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

func (suite *TaskRepositoryTestSuite) TestListTasks() {
	// Create multiple tasks for the same user
	for i := 1; i <= 5; i++ {
		task := &model.Task{
			UserID:      suite.userID,
			Title:       "Test Task " + string(rune('0'+i)),
			Description: "Test Description " + string(rune('0'+i)),
			Status:      model.StatusTodo,
			Priority:    model.PriorityMedium,
		}
		_, err := suite.repo.Create(suite.ctx, task)
		assert.NoError(suite.T(), err)
	}

	// Create a task for a different user
	otherTask := &model.Task{
		UserID:      "other-user-id",
		Title:       "Other User Task",
		Description: "Other User Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
	}
	_, err := suite.repo.Create(suite.ctx, otherTask)
	assert.NoError(suite.T(), err)

	// List all tasks without filter
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

	// List tasks with status filter
	status := string(model.StatusTodo)
	statusFilter := &repository.TaskFilter{
		Status: &status,
	}
	statusTasks, statusTotal, err := suite.repo.List(suite.ctx, statusFilter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(6), statusTotal)
	assert.Len(suite.T(), statusTasks, 6)
}

func (suite *TaskRepositoryTestSuite) TestListTasksByUser() {
	// Create multiple tasks for the same user
	for i := 1; i <= 5; i++ {
		task := &model.Task{
			UserID:      suite.userID,
			Title:       "Test Task " + string(rune('0'+i)),
			Description: "Test Description " + string(rune('0'+i)),
			Status:      model.StatusTodo,
			Priority:    model.PriorityMedium,
		}
		_, err := suite.repo.Create(suite.ctx, task)
		assert.NoError(suite.T(), err)
	}

	// List tasks by user
	filter := &repository.TaskFilter{}
	tasks, total, err := suite.repo.ListByUser(suite.ctx, suite.userID, filter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(5), total)
	assert.Len(suite.T(), tasks, 5)

	// Test pagination
	tasksPage1, totalPage1, err := suite.repo.ListByUser(suite.ctx, suite.userID, filter, 1, 2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(5), totalPage1)
	assert.Len(suite.T(), tasksPage1, 2)

	tasksPage2, totalPage2, err := suite.repo.ListByUser(suite.ctx, suite.userID, filter, 2, 2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(5), totalPage2)
	assert.Len(suite.T(), tasksPage2, 2)
}

func (suite *TaskRepositoryTestSuite) TestListTasksWithSorting() {
	// Create tasks with different priorities
	priorities := []model.TaskPriority{
		model.PriorityLow,
		model.PriorityMedium,
		model.PriorityHigh,
		model.PriorityUrgent,
	}
	
	for i, priority := range priorities {
		task := &model.Task{
			UserID:      suite.userID,
			Title:       "Test Task " + string(rune('A'+i)),
			Description: "Test Description",
			Status:      model.StatusTodo,
			Priority:    priority,
		}
		_, err := suite.repo.Create(suite.ctx, task)
		assert.NoError(suite.T(), err)
	}

	// Test sorting by priority ascending
	ascFilter := &repository.TaskFilter{
		SortBy:   "priority",
		SortDesc: false,
	}
	ascTasks, _, err := suite.repo.ListByUser(suite.ctx, suite.userID, ascFilter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), model.PriorityLow, ascTasks[0].Priority)
	assert.Equal(suite.T(), model.PriorityMedium, ascTasks[1].Priority)

	// Test sorting by priority descending
	descFilter := &repository.TaskFilter{
		SortBy:   "priority",
		SortDesc: true,
	}
	descTasks, _, err := suite.repo.ListByUser(suite.ctx, suite.userID, descFilter, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), model.PriorityUrgent, descTasks[0].Priority)
	assert.Equal(suite.T(), model.PriorityHigh, descTasks[1].Priority)
}

func TestTaskRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(TaskRepositoryTestSuite))
}