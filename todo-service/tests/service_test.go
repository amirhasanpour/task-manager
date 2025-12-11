package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
	"github.com/amirhasanpour/task-manager/todo-service/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Mock repositories
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

// Mock cache
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

// Mock metrics
type MockMetricsCollector struct {
	mock.Mock
}

func (m *MockMetricsCollector) UpdateTasksCount(count int) {
	m.Called(count)
}

func (m *MockMetricsCollector) UpdateTasksCountByStatus(status string, count int) {
	m.Called(status, count)
}

func (m *MockMetricsCollector) UpdateTasksCountByPriority(priority string, count int) {
	m.Called(priority, count)
}

func (m *MockMetricsCollector) IncrementCacheHits() {
	m.Called()
}

func (m *MockMetricsCollector) IncrementCacheMisses() {
	m.Called()
}

func (m *MockMetricsCollector) IncrementDatabaseErrors() {
	m.Called()
}

func (m *MockMetricsCollector) IncrementCacheErrors() {
	m.Called()
}

func (m *MockMetricsCollector) IncrementValidationErrors() {
	m.Called()
}

type TaskServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	repo        *MockTaskRepository
	cache       *MockTaskCache
	metrics     *MockMetricsCollector
	service     service.TaskService
	testUserID  string
	testTaskID  string
}

func (suite *TaskServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.repo = new(MockTaskRepository)
	suite.cache = new(MockTaskCache)
	suite.metrics = new(MockMetricsCollector)
	suite.service = service.NewTaskService(suite.repo, suite.cache, (*service.MetricsCollector)(nil))
	suite.testUserID = "test-user-id"
	suite.testTaskID = "test-task-id"
	
	// Initialize logger for tests
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func (suite *TaskServiceTestSuite) TearDownTest() {
	suite.repo.AssertExpectations(suite.T())
	suite.cache.AssertExpectations(suite.T())
	suite.metrics.AssertExpectations(suite.T())
}

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

	// Expectations
	suite.metrics.On("IncrementValidationErrors").Maybe()
	suite.metrics.On("IncrementDatabaseErrors").Maybe()
	suite.metrics.On("IncrementCacheErrors").Maybe()
	suite.metrics.On("UpdateTasksCountByStatus", "TODO", 1).Once()
	suite.metrics.On("UpdateTasksCountByPriority", "MEDIUM", 1).Once()
	
	suite.repo.On("Create", suite.ctx, mock.AnythingOfType("*model.Task")).
		Return(expectedTask, nil).
		Once()
	
	suite.cache.On("InvalidateUserTasks", suite.ctx, suite.testUserID).
		Return(nil).
		Once()

	// Test
	task, err := suite.service.CreateTask(suite.ctx, req)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), suite.testTaskID, task.ID)
	assert.Equal(suite.T(), "Test Task", task.Title)
}

func (suite *TaskServiceTestSuite) TestCreateTask_ValidationError() {
	req := &service.CreateTaskRequest{
		UserID: "", // Empty user ID should fail validation
		Title:  "Test Task",
	}

	// Expectations
	suite.metrics.On("IncrementValidationErrors").Once()

	// Test
	task, err := suite.service.CreateTask(suite.ctx, req)
	
	// Assertions
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), task)
	
	// Check it's a validation error
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.InvalidArgument, st.Code())
}

func (suite *TaskServiceTestSuite) TestGetTask_CacheHit() {
	expectedTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  "Test Task",
	}

	// Expectations - cache hit
	suite.metrics.On("IncrementCacheHits").Once()
	suite.metrics.On("IncrementCacheMisses").Maybe()
	suite.metrics.On("IncrementDatabaseErrors").Maybe()
	suite.metrics.On("IncrementCacheErrors").Maybe()
	
	suite.cache.On("GetTask", suite.ctx, suite.testTaskID).
		Return(expectedTask, nil).
		Once()

	// Test
	task, err := suite.service.GetTask(suite.ctx, suite.testTaskID)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), suite.testTaskID, task.ID)
	assert.Equal(suite.T(), "Test Task", task.Title)
	
	// Repository should not be called for cache hit
	suite.repo.AssertNotCalled(suite.T(), "FindByID", mock.Anything, mock.Anything)
}

func (suite *TaskServiceTestSuite) TestGetTask_CacheMiss() {
	expectedTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  "Test Task",
	}

	// Expectations - cache miss
	suite.metrics.On("IncrementCacheHits").Maybe()
	suite.metrics.On("IncrementCacheMisses").Once()
	suite.metrics.On("IncrementDatabaseErrors").Maybe()
	suite.metrics.On("IncrementCacheErrors").Maybe()
	
	suite.cache.On("GetTask", suite.ctx, suite.testTaskID).
		Return(nil, nil). // Cache miss
		Once()
	
	suite.repo.On("FindByID", suite.ctx, suite.testTaskID).
		Return(expectedTask, nil).
		Once()
	
	suite.cache.On("SetTask", suite.ctx, expectedTask).
		Return(nil).
		Once()

	// Test
	task, err := suite.service.GetTask(suite.ctx, suite.testTaskID)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), suite.testTaskID, task.ID)
}

func (suite *TaskServiceTestSuite) TestUpdateTask_Success() {
	title := "Updated Title"
	status := "IN_PROGRESS"
	priority := "HIGH"
	
	req := &service.UpdateTaskRequest{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  &title,
		Status: &status,
		Priority: &priority,
	}

	existingTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  "Original Title",
		Status: model.StatusTodo,
		Priority: model.PriorityMedium,
	}

	updatedTask := &model.Task{
		ID:     suite.testTaskID,
		UserID: suite.testUserID,
		Title:  "Updated Title",
		Status: model.StatusInProgress,
		Priority: model.PriorityHigh,
	}

	// Expectations
	suite.metrics.On("IncrementDatabaseErrors").Maybe()
	suite.metrics.On("IncrementCacheErrors").Maybe()
	suite.metrics.On("UpdateTasksCountByStatus", "TODO", -1).Once()
	suite.metrics.On("UpdateTasksCountByStatus", "IN_PROGRESS", 1).Once()
	suite.metrics.On("UpdateTasksCountByPriority", "MEDIUM", -1).Once()
	suite.metrics.On("UpdateTasksCountByPriority", "HIGH", 1).Once()
	
	suite.repo.On("FindByIDAndUser", suite.ctx, suite.testTaskID, suite.testUserID).
		Return(existingTask, nil).
		Once()
	
	suite.repo.On("Update", suite.ctx, mock.AnythingOfType("*model.Task")).
		Return(updatedTask, nil).
		Once()
	
	suite.cache.On("SetTask", suite.ctx, updatedTask).
		Return(nil).
		Once()
	
	suite.cache.On("InvalidateUserTasks", suite.ctx, suite.testUserID).
		Return(nil).
		Once()

	// Test
	task, err := suite.service.UpdateTask(suite.ctx, req)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), task)
	assert.Equal(suite.T(), "Updated Title", task.Title)
	assert.Equal(suite.T(), model.StatusInProgress, task.Status)
	assert.Equal(suite.T(), model.PriorityHigh, task.Priority)
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

	// Expectations
	suite.metrics.On("IncrementCacheHits").Maybe()
	suite.metrics.On("IncrementCacheMisses").Once()
	suite.metrics.On("IncrementDatabaseErrors").Maybe()
	suite.metrics.On("IncrementCacheErrors").Maybe()
	
	suite.cache.On("GetTasksList", suite.ctx, mock.AnythingOfType("string")).
		Return([]*model.Task(nil), int64(0), nil). // Cache miss
		Once()
	
	suite.repo.On("ListByUser", suite.ctx, suite.testUserID, filter, 1, 10).
		Return(tasks, total, nil).
		Once()
	
	suite.cache.On("SetTasksList", suite.ctx, mock.AnythingOfType("string"), tasks, total).
		Return(nil).
		Once()

	// Test
	resultTasks, resultTotal, err := suite.service.ListTasksByUser(suite.ctx, suite.testUserID, filter, 1, 10)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), resultTasks, 2)
	assert.Equal(suite.T(), total, resultTotal)
	assert.Equal(suite.T(), "Task 1", resultTasks[0].Title)
	assert.Equal(suite.T(), "Task 2", resultTasks[1].Title)
}

func stringPtr(s string) *string {
	return &s
}

func TestTaskServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TaskServiceTestSuite))
}