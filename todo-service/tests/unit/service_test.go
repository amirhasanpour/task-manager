package tests

import (
	"context"
	"testing"
	"time"

	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
	"github.com/amirhasanpour/task-manager/todo-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ==================== MOCKS ====================

type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *model.Task) (*model.Task, error) {
	args := m.Called(ctx, task)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByID(ctx context.Context, id string) (*model.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByIDAndUser(ctx context.Context, id, userID string) (*model.Task, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskRepository) Update(ctx context.Context, task *model.Task) (*model.Task, error) {
	args := m.Called(ctx, task)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskRepository) DeleteByUser(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockTaskRepository) List(ctx context.Context, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	args := m.Called(ctx, filter, page, pageSize)
	return args.Get(0).([]*model.Task), args.Get(1).(int64), args.Error(2)
}

func (m *MockTaskRepository) ListByUser(ctx context.Context, userID string, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	args := m.Called(ctx, userID, filter, page, pageSize)
	return args.Get(0).([]*model.Task), args.Get(1).(int64), args.Error(2)
}

type MockTaskCache struct {
	mock.Mock
}

func (m *MockTaskCache) GetTask(ctx context.Context, id string) (*model.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskCache) SetTask(ctx context.Context, task *model.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskCache) DeleteTask(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskCache) GetTasksList(ctx context.Context, key string) ([]*model.Task, int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]*model.Task), args.Get(1).(int64), args.Error(2)
}

func (m *MockTaskCache) SetTasksList(ctx context.Context, key string, tasks []*model.Task, total int64) error {
	args := m.Called(ctx, key, tasks, total)
	return args.Error(0)
}

func (m *MockTaskCache) DeleteTasksList(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockTaskCache) InvalidateUserTasks(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// ==================== TEST SUITE ====================

type TaskServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	repo        *MockTaskRepository
	cache       *MockTaskCache
	service     service.TaskService
	testUserID  string
	testTaskID  string
	
	// Track metrics calls
	metricsCalls struct {
		updateTasksCount            int
		updateTasksCountByStatus    map[string]int
		updateTasksCountByPriority  map[string]int
		cacheHits                   int
		cacheMisses                 int
		databaseErrors              int
		cacheErrors                 int
		validationErrors            int
	}
}

func (suite *TaskServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.repo = new(MockTaskRepository)
	suite.cache = new(MockTaskCache)
	
	// Initialize metrics tracking
	suite.metricsCalls = struct {
		updateTasksCount            int
		updateTasksCountByStatus    map[string]int
		updateTasksCountByPriority  map[string]int
		cacheHits                   int
		cacheMisses                 int
		databaseErrors              int
		cacheErrors                 int
		validationErrors            int
	}{
		updateTasksCountByStatus:   make(map[string]int),
		updateTasksCountByPriority: make(map[string]int),
	}
	
	// Create a metrics collector with tracking functions
	metricsCollector := service.NewMetricsCollector(
		// updateTasksCount
		func(count int) {
			suite.metricsCalls.updateTasksCount = count
		},
		// updateTasksCountByStatus
		func(status string, count int) {
			suite.metricsCalls.updateTasksCountByStatus[status] = count
		},
		// updateTasksCountByPriority
		func(priority string, count int) {
			suite.metricsCalls.updateTasksCountByPriority[priority] = count
		},
		// incrementCacheHits
		func() {
			suite.metricsCalls.cacheHits++
		},
		// incrementCacheMisses
		func() {
			suite.metricsCalls.cacheMisses++
		},
		// incrementDatabaseErrors
		func() {
			suite.metricsCalls.databaseErrors++
		},
		// incrementCacheErrors
		func() {
			suite.metricsCalls.cacheErrors++
		},
		// incrementValidationErrors
		func() {
			suite.metricsCalls.validationErrors++
		},
	)
	
	suite.service = service.NewTaskService(suite.repo, suite.cache, metricsCollector)
	suite.testUserID = "test-user-123"
	suite.testTaskID = "test-task-456"
}

func (suite *TaskServiceTestSuite) TearDownTest() {
	suite.repo.AssertExpectations(suite.T())
	suite.cache.AssertExpectations(suite.T())
}

// ==================== TEST CASES ====================

func (suite *TaskServiceTestSuite) TestCreateTask_Success() {
	dueDate := time.Now().Add(24 * time.Hour)
	
	req := &service.CreateTaskRequest{
		UserID:      suite.testUserID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "TODO",
		Priority:    "MEDIUM",
		DueDate:     &dueDate,
	}

	expectedTask := &model.Task{
		ID:          suite.testTaskID,
		UserID:      suite.testUserID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
		DueDate:     &dueDate,
	}

	// Setup expectations - use mock.AnythingOfType("*context.valueCtx") for context
	suite.repo.On("Create", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.Task")).
		Return(expectedTask, nil).
		Once()
	
	suite.cache.On("InvalidateUserTasks", mock.AnythingOfType("*context.valueCtx"), suite.testUserID).
		Return(nil).
		Once()
	
	suite.cache.On("SetTask", mock.AnythingOfType("*context.valueCtx"), expectedTask).
		Return(nil).
		Once()

	// Execute
	task, err := suite.service.CreateTask(suite.ctx, req)

	// Verify
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), suite.testTaskID, task.ID)
	assert.Equal(suite.T(), "Test Task", task.Title)
	
	// Verify metrics were called
	assert.Equal(suite.T(), 1, suite.metricsCalls.updateTasksCountByStatus["TODO"])
	assert.Equal(suite.T(), 1, suite.metricsCalls.updateTasksCountByPriority["MEDIUM"])
}

func (suite *TaskServiceTestSuite) TestCreateTask_ValidationError_EmptyTitle() {
	req := &service.CreateTaskRequest{
		UserID: suite.testUserID,
		Title:  "", // Empty title should fail
	}

	// Execute
	task, err := suite.service.CreateTask(suite.ctx, req)

	// Verify
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), task)
	assert.Contains(suite.T(), err.Error(), "title is required")
	
	// Verify validation error metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.validationErrors)
}

func (suite *TaskServiceTestSuite) TestCreateTask_ValidationError_InvalidStatus() {
	req := &service.CreateTaskRequest{
		UserID: suite.testUserID,
		Title:  "Test Task",
		Status: "INVALID_STATUS", // Invalid status
	}

	// Execute
	task, err := suite.service.CreateTask(suite.ctx, req)

	// Verify
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), task)
	assert.Contains(suite.T(), err.Error(), "status must be one of")
	
	// Verify validation error metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.validationErrors)
}

func (suite *TaskServiceTestSuite) TestGetTask_CacheHit() {
	expectedTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  "Cached Task",
	}

	// Setup expectations - cache hit
	suite.cache.On("GetTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(expectedTask, nil).
		Once()

	// Execute
	task, err := suite.service.GetTask(suite.ctx, suite.testTaskID)

	// Verify
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), "Cached Task", task.Title)
	
	// Verify cache hit metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.cacheHits)
	
	// Repository should NOT be called for cache hit
	suite.repo.AssertNotCalled(suite.T(), "FindByID", mock.Anything, mock.Anything)
}

func (suite *TaskServiceTestSuite) TestGetTask_CacheMiss() {
	expectedTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  "Database Task",
	}

	// Setup expectations - cache miss
	suite.cache.On("GetTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil, nil). // Cache miss
		Once()
	
	suite.repo.On("FindByID", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(expectedTask, nil).
		Once()
	
	suite.cache.On("SetTask", mock.AnythingOfType("*context.valueCtx"), expectedTask).
		Return(nil).
		Once()

	// Execute
	task, err := suite.service.GetTask(suite.ctx, suite.testTaskID)

	// Verify
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), "Database Task", task.Title)
	
	// Verify cache miss metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.cacheMisses)
}

func (suite *TaskServiceTestSuite) TestGetTask_NotFound() {
	// Setup expectations
	suite.cache.On("GetTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil, nil).
		Once()
	
	suite.repo.On("FindByID", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil, nil). // Not found
		Once()

	// Execute
	task, err := suite.service.GetTask(suite.ctx, suite.testTaskID)

	// Verify
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), task)
	assert.Contains(suite.T(), err.Error(), "not found")
	
	// Verify cache miss metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.cacheMisses)
}

func (suite *TaskServiceTestSuite) TestGetTaskByUser_Success() {
	expectedTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  "User Task",
	}

	// Setup expectations
	suite.cache.On("GetTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(expectedTask, nil). // Cache hit
		Once()

	// Execute
	task, err := suite.service.GetTaskByUser(suite.ctx, suite.testTaskID, suite.testUserID)

	// Verify
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), "User Task", task.Title)
	assert.Equal(suite.T(), suite.testUserID, task.UserID)
	
	// Verify cache hit metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.cacheHits)
}

func (suite *TaskServiceTestSuite) TestGetTaskByUser_WrongUser() {
	expectedTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: "different-user", // Different user ID
		Title:  "Wrong User Task",
	}

	// Setup expectations - cache hit but wrong user
	// Only cache.GetTask should be called
	suite.cache.On("GetTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(expectedTask, nil). // Cache hit but wrong user
		Once()

	// Execute
	task, err := suite.service.GetTaskByUser(suite.ctx, suite.testTaskID, suite.testUserID)

	// Verify
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), task)
	
	// Check for PermissionDenied error
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.PermissionDenied, st.Code())
	assert.Contains(suite.T(), st.Message(), "task not found")
	
	// IMPORTANT: Verify cache hit metric was NOT incremented
	// The service should NOT increment cache hits when user doesn't match
	// Check if this is the actual behavior in your service code
	// If service increments hits before checking user, we need to expect 1
	// If service doesn't increment hits, we expect 0
	
	// Based on the service code, it increments cacheHits in the else if block
	// But when user doesn't match, it returns early, so cacheHits should be 0
	assert.Equal(suite.T(), 0, suite.metricsCalls.cacheHits)
}

func (suite *TaskServiceTestSuite) TestUpdateTask_Success() {
	title := "Updated Title"
	status := "IN_PROGRESS"
	priority := "HIGH"
	
	req := &service.UpdateTaskRequest{
		ID:          suite.testTaskID,
		UserID:      suite.testUserID,
		Title:       &title,
		Status:      &status,
		Priority:    &priority,
	}

	existingTask := &model.Task{
		ID:          suite.testTaskID,
		UserID:      suite.testUserID,
		Title:       "Original Title",
		Description: "Original Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	updatedTask := &model.Task{
		ID:          suite.testTaskID,
		UserID:      suite.testUserID,
		Title:       "Updated Title",
		Description: "Original Description",
		Status:      model.StatusInProgress,
		Priority:    model.PriorityHigh,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Setup expectations
	suite.repo.On("FindByIDAndUser", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID, suite.testUserID).
		Return(existingTask, nil).
		Once()
	
	suite.repo.On("Update", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.Task")).
		Return(updatedTask, nil).
		Once()
	
	suite.cache.On("SetTask", mock.AnythingOfType("*context.valueCtx"), updatedTask).
		Return(nil).
		Once()
	
	suite.cache.On("InvalidateUserTasks", mock.AnythingOfType("*context.valueCtx"), suite.testUserID).
		Return(nil).
		Once()

	// Execute
	task, err := suite.service.UpdateTask(suite.ctx, req)

	// Verify
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), "Updated Title", task.Title)
	assert.Equal(suite.T(), model.StatusInProgress, task.Status)
	assert.Equal(suite.T(), model.PriorityHigh, task.Priority)
	
	// Verify metrics were updated for status and priority changes
	assert.Equal(suite.T(), -1, suite.metricsCalls.updateTasksCountByStatus["TODO"])
	assert.Equal(suite.T(), 1, suite.metricsCalls.updateTasksCountByStatus["IN_PROGRESS"])
	assert.Equal(suite.T(), -1, suite.metricsCalls.updateTasksCountByPriority["MEDIUM"])
	assert.Equal(suite.T(), 1, suite.metricsCalls.updateTasksCountByPriority["HIGH"])
}

func (suite *TaskServiceTestSuite) TestUpdateTask_TaskNotFound() {
	title := "Updated Title"
	
	req := &service.UpdateTaskRequest{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  &title,
	}

	// Setup expectations
	suite.repo.On("FindByIDAndUser", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID, suite.testUserID).
		Return(nil, nil). // Task not found
		Once()

	// Execute
	task, err := suite.service.UpdateTask(suite.ctx, req)

	// Verify
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), task)
	assert.Contains(suite.T(), err.Error(), "not found")
}

func (suite *TaskServiceTestSuite) TestUpdateTask_PartialUpdate() {
	title := "Updated Title Only"
	
	req := &service.UpdateTaskRequest{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  &title,
		// Status and Priority not provided - should remain unchanged
	}

	existingTask := &model.Task{
		ID:          suite.testTaskID,
		UserID:      suite.testUserID,
		Title:       "Original Title",
		Description: "Original Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
	}

	updatedTask := &model.Task{
		ID:          suite.testTaskID,
		UserID:      suite.testUserID,
		Title:       "Updated Title Only",
		Description: "Original Description",
		Status:      model.StatusTodo,     // Unchanged
		Priority:    model.PriorityMedium, // Unchanged
	}

	// Setup expectations
	suite.repo.On("FindByIDAndUser", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID, suite.testUserID).
		Return(existingTask, nil).
		Once()
	
	suite.repo.On("Update", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.Task")).
		Return(updatedTask, nil).
		Once()
	
	suite.cache.On("SetTask", mock.AnythingOfType("*context.valueCtx"), updatedTask).
		Return(nil).
		Once()
	
	suite.cache.On("InvalidateUserTasks", mock.AnythingOfType("*context.valueCtx"), suite.testUserID).
		Return(nil).
		Once()

	// Execute
	task, err := suite.service.UpdateTask(suite.ctx, req)

	// Verify
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), "Updated Title Only", task.Title)
	assert.Equal(suite.T(), model.StatusTodo, task.Status) // Should remain unchanged
	assert.Equal(suite.T(), model.PriorityMedium, task.Priority) // Should remain unchanged
	
	// Verify metrics were NOT updated (status and priority didn't change)
	assert.Equal(suite.T(), 0, suite.metricsCalls.updateTasksCountByStatus["TODO"])
	assert.Equal(suite.T(), 0, suite.metricsCalls.updateTasksCountByPriority["MEDIUM"])
}

func (suite *TaskServiceTestSuite) TestDeleteTask_Success() {
	task := &model.Task{
		ID:       suite.testTaskID,
		UserID:   suite.testUserID,
		Title:    "Task to Delete",
		Status:   model.StatusTodo,
		Priority: model.PriorityMedium,
	}

	// Setup expectations
	suite.repo.On("FindByID", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(task, nil).
		Once()
	
	suite.repo.On("Delete", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil).
		Once()
	
	suite.cache.On("DeleteTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil).
		Once()
	
	suite.cache.On("InvalidateUserTasks", mock.AnythingOfType("*context.valueCtx"), suite.testUserID).
		Return(nil).
		Once()

	// Execute
	err := suite.service.DeleteTask(suite.ctx, suite.testTaskID)

	// Verify
	assert.NoError(suite.T(), err)
	
	// Verify metrics were updated
	assert.Equal(suite.T(), -1, suite.metricsCalls.updateTasksCountByStatus["TODO"])
	assert.Equal(suite.T(), -1, suite.metricsCalls.updateTasksCountByPriority["MEDIUM"])
}

func (suite *TaskServiceTestSuite) TestDeleteTaskByUser_Success() {
	task := &model.Task{
		ID:       suite.testTaskID,
		UserID:   suite.testUserID,
		Title:    "Task to Delete",
		Status:   model.StatusDone,
		Priority: model.PriorityHigh,
	}

	// Setup expectations
	suite.repo.On("FindByIDAndUser", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID, suite.testUserID).
		Return(task, nil).
		Once()
	
	suite.repo.On("DeleteByUser", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID, suite.testUserID).
		Return(nil).
		Once()
	
	suite.cache.On("DeleteTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil).
		Once()
	
	suite.cache.On("InvalidateUserTasks", mock.AnythingOfType("*context.valueCtx"), suite.testUserID).
		Return(nil).
		Once()

	// Execute
	err := suite.service.DeleteTaskByUser(suite.ctx, suite.testTaskID, suite.testUserID)

	// Verify
	assert.NoError(suite.T(), err)
	
	// Verify metrics were updated
	assert.Equal(suite.T(), -1, suite.metricsCalls.updateTasksCountByStatus["DONE"])
	assert.Equal(suite.T(), -1, suite.metricsCalls.updateTasksCountByPriority["HIGH"])
}

func (suite *TaskServiceTestSuite) TestListTasks_Success() {
	filter := &repository.TaskFilter{
		Status: stringPtr("TODO"),
		SortBy: "created_at",
		SortDesc: true,
	}
	
	tasks := []*model.Task{
		{
			ID:     "task-1",
			UserID: suite.testUserID,
			Title:  "Task 1",
			Status: model.StatusTodo,
			Priority: model.PriorityMedium,
		},
		{
			ID:     "task-2",
			UserID: suite.testUserID,
			Title:  "Task 2",
			Status: model.StatusTodo,
			Priority: model.PriorityHigh,
		},
	}
	
	const total int64 = 2

	// Setup expectations
	suite.cache.On("GetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).
		Return([]*model.Task(nil), int64(0), nil). // Cache miss
		Once()
	
	suite.repo.On("List", mock.AnythingOfType("*context.valueCtx"), filter, 1, 10).
		Return(tasks, total, nil).
		Once()
	
	suite.cache.On("SetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), tasks, total).
		Return(nil).
		Once()

	// Execute
	resultTasks, resultTotal, err := suite.service.ListTasks(suite.ctx, filter, 1, 10)

	// Verify
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), resultTasks, 2)
	assert.Equal(suite.T(), total, resultTotal)
	assert.Equal(suite.T(), "Task 1", resultTasks[0].Title)
	assert.Equal(suite.T(), "Task 2", resultTasks[1].Title)
	
	// Verify cache miss metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.cacheMisses)
}

func (suite *TaskServiceTestSuite) TestListTasks_CacheHit() {
	tasks := []*model.Task{
		{
			ID:     "task-1",
			UserID: suite.testUserID,
			Title:  "Cached Task",
		},
	}
	
	const total int64 = 1

	// Setup expectations - cache hit
	suite.cache.On("GetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).
		Return(tasks, total, nil). // Cache hit
		Once()

	// Execute
	resultTasks, resultTotal, err := suite.service.ListTasks(suite.ctx, nil, 1, 10)

	// Verify
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), resultTasks, 1)
	assert.Equal(suite.T(), total, resultTotal)
	assert.Equal(suite.T(), "Cached Task", resultTasks[0].Title)
	
	// Verify cache hit metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.cacheHits)
	
	// Repository should NOT be called for cache hit
	suite.repo.AssertNotCalled(suite.T(), "List", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *TaskServiceTestSuite) TestListTasksByUser_Success() {
	filter := &repository.TaskFilter{
		Status: stringPtr("TODO"),
		SortBy: "created_at",
		SortDesc: true,
	}
	
	tasks := []*model.Task{
		{
			ID:     "task-1",
			UserID: suite.testUserID,
			Title:  "User Task 1",
			Status: model.StatusTodo,
			Priority: model.PriorityMedium,
		},
		{
			ID:     "task-2",
			UserID: suite.testUserID,
			Title:  "User Task 2",
			Status: model.StatusTodo,
			Priority: model.PriorityHigh,
		},
	}
	
	const total int64 = 2

	// Setup expectations
	suite.cache.On("GetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).
		Return([]*model.Task(nil), int64(0), nil). // Cache miss
		Once()
	
	suite.repo.On("ListByUser", mock.AnythingOfType("*context.valueCtx"), suite.testUserID, filter, 1, 10).
		Return(tasks, total, nil).
		Once()
	
	suite.cache.On("SetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), tasks, total).
		Return(nil).
		Once()

	// Execute
	resultTasks, resultTotal, err := suite.service.ListTasksByUser(suite.ctx, suite.testUserID, filter, 1, 10)

	// Verify
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), resultTasks, 2)
	assert.Equal(suite.T(), total, resultTotal)
	assert.Equal(suite.T(), "User Task 1", resultTasks[0].Title)
	assert.Equal(suite.T(), "User Task 2", resultTasks[1].Title)
	
	// Verify cache miss metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.cacheMisses)
}

func (suite *TaskServiceTestSuite) TestListTasksByUser_CacheHit() {
	tasks := []*model.Task{
		{
			ID:     "task-1",
			UserID: suite.testUserID,
			Title:  "Cached User Task",
		},
	}
	
	const total int64 = 1

	// Setup expectations - cache hit
	suite.cache.On("GetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).
		Return(tasks, total, nil). // Cache hit
		Once()

	// Execute
	resultTasks, resultTotal, err := suite.service.ListTasksByUser(suite.ctx, suite.testUserID, nil, 1, 10)

	// Verify
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), resultTasks, 1)
	assert.Equal(suite.T(), total, resultTotal)
	assert.Equal(suite.T(), "Cached User Task", resultTasks[0].Title)
	
	// Verify cache hit metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.cacheHits)
	
	// Repository should NOT be called for cache hit
	suite.repo.AssertNotCalled(suite.T(), "ListByUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *TaskServiceTestSuite) TestListTasks_Pagination() {
	tasks := []*model.Task{
		{
			ID:     "task-1",
			UserID: suite.testUserID,
			Title:  "Task 1",
		},
	}
	
	const total int64 = 3

	// Setup expectations - page 1
	suite.cache.On("GetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).
		Return([]*model.Task(nil), int64(0), nil). // Cache miss
		Once()
	
	suite.repo.On("List", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*repository.TaskFilter"), 1, 2).
		Return(tasks, total, nil).
		Once()
	
	suite.cache.On("SetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), tasks, total).
		Return(nil).
		Once()

	// Execute - page 1, size 2
	resultTasks, resultTotal, err := suite.service.ListTasks(suite.ctx, nil, 1, 2)

	// Verify
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), resultTasks, 1)
	assert.Equal(suite.T(), total, resultTotal)
}

func (suite *TaskServiceTestSuite) TestListTasks_PageValidation() {
	// Test page < 1 should default to 1
	suite.cache.On("GetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).
		Return([]*model.Task(nil), int64(0), nil).
		Once()
	
	suite.repo.On("List", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*repository.TaskFilter"), 1, 10). // Should use page 1
		Return([]*model.Task{}, int64(0), nil).
		Once()
	
	suite.cache.On("SetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), []*model.Task{}, int64(0)).
		Return(nil).
		Once()

	_, _, err := suite.service.ListTasks(suite.ctx, nil, 0, 10) // Page 0
	assert.NoError(suite.T(), err)
}

func (suite *TaskServiceTestSuite) TestListTasks_PageSizeValidation() {
	// Test pageSize > 100 should default to 100
	suite.cache.On("GetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).
		Return([]*model.Task(nil), int64(0), nil).
		Once()
	
	suite.repo.On("List", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*repository.TaskFilter"), 1, 100). // Should use size 100
		Return([]*model.Task{}, int64(0), nil).
		Once()
	
	suite.cache.On("SetTasksList", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), []*model.Task{}, int64(0)).
		Return(nil).
		Once()

	_, _, err := suite.service.ListTasks(suite.ctx, nil, 1, 150) // Size 150
	assert.NoError(suite.T(), err)
}

func (suite *TaskServiceTestSuite) TestCacheErrorHandling() {
	// Test that cache errors don't fail the operation
	expectedTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  "Task",
	}

	// Setup expectations - cache error
	suite.cache.On("GetTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil, assert.AnError). // Cache error
		Once()
	
	suite.repo.On("FindByID", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(expectedTask, nil).
		Once()
	
	suite.cache.On("SetTask", mock.AnythingOfType("*context.valueCtx"), expectedTask).
		Return(assert.AnError). // Cache error on set
		Once()

	// Execute
	task, err := suite.service.GetTask(suite.ctx, suite.testTaskID)

	// Verify
	assert.NoError(suite.T(), err) // Should not fail even with cache errors
	assert.NotNil(suite.T(), task)
	
	// Verify cache error metric was incremented
	assert.Equal(suite.T(), 2, suite.metricsCalls.cacheErrors) // One for get, one for set
}

func (suite *TaskServiceTestSuite) TestDatabaseErrorHandling() {
	// Setup expectations - database error
	suite.cache.On("GetTask", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil, nil). // Cache miss
		Once()
	
	suite.repo.On("FindByID", mock.AnythingOfType("*context.valueCtx"), suite.testTaskID).
		Return(nil, assert.AnError). // Database error
		Once()

	// Execute
	task, err := suite.service.GetTask(suite.ctx, suite.testTaskID)

	// Verify
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), task)
	
	// Verify database error metric was incremented
	assert.Equal(suite.T(), 1, suite.metricsCalls.databaseErrors)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

func TestTaskServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TaskServiceTestSuite))
}