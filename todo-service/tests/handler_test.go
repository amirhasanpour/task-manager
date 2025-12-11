package tests

import (
	"context"
	"testing"
	"time"

	"github.com/amirhasanpour/task-manager/todo-service/internal/handler"
	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
	"github.com/amirhasanpour/task-manager/todo-service/internal/service"
	pb "github.com/amirhasanpour/task-manager/todo-service/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Mock service
type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) CreateTask(ctx context.Context, req *service.CreateTaskRequest) (*model.Task, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskService) GetTaskByUser(ctx context.Context, id, userID string) (*model.Task, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskService) UpdateTask(ctx context.Context, req *service.UpdateTaskRequest) (*model.Task, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskService) DeleteTask(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskService) DeleteTaskByUser(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockTaskService) ListTasks(ctx context.Context, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	args := m.Called(ctx, filter, page, pageSize)
	return args.Get(0).([]*model.Task), args.Get(1).(int64), args.Error(2)
}

func (m *MockTaskService) ListTasksByUser(ctx context.Context, userID string, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	args := m.Called(ctx, userID, filter, page, pageSize)
	return args.Get(0).([]*model.Task), args.Get(1).(int64), args.Error(2)
}

type TaskHandlerTestSuite struct {
	suite.Suite
	ctx      context.Context
	service  *MockTaskService
	handler  *handler.TaskHandler
	userID   string
	taskID   string
}

func (suite *TaskHandlerTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.service = new(MockTaskService)
	suite.handler = handler.NewTaskHandler(suite.service)
	suite.userID = "test-user-id"
	suite.taskID = "test-task-id"
	
	// Initialize logger for tests
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func (suite *TaskHandlerTestSuite) TestCreateTask_Success() {
	dueDate := time.Now().Add(24 * time.Hour)
	
	req := &pb.CreateTaskRequest{
		UserId:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      pb.TaskStatus_TODO,
		Priority:    pb.TaskPriority_MEDIUM,
		DueDate:     timestamppb.New(dueDate),
	}

	expectedTask := &model.Task{
		ID:          suite.taskID,
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
		DueDate:     &dueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Expectations
	suite.service.On("CreateTask", suite.ctx, mock.AnythingOfType("*service.CreateTaskRequest")).
		Return(expectedTask, nil).
		Once()

	// Test
	resp, err := suite.handler.CreateTask(suite.ctx, req)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotNil(suite.T(), resp.Task)
	assert.Equal(suite.T(), suite.taskID, resp.Task.Id)
	assert.Equal(suite.T(), "Test Task", resp.Task.Title)
	assert.Equal(suite.T(), suite.userID, resp.Task.UserId)
	assert.Equal(suite.T(), pb.TaskStatus_TODO, resp.Task.Status)
	assert.Equal(suite.T(), pb.TaskPriority_MEDIUM, resp.Task.Priority)
}

func (suite *TaskHandlerTestSuite) TestGetTask_Success() {
	req := &pb.GetTaskRequest{
		Id: suite.taskID,
	}

	expectedTask := &model.Task{
		ID:          suite.taskID,
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Expectations
	suite.service.On("GetTask", suite.ctx, suite.taskID).
		Return(expectedTask, nil).
		Once()

	// Test
	resp, err := suite.handler.GetTask(suite.ctx, req)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotNil(suite.T(), resp.Task)
	assert.Equal(suite.T(), suite.taskID, resp.Task.Id)
	assert.Equal(suite.T(), "Test Task", resp.Task.Title)
}

func (suite *TaskHandlerTestSuite) TestUpdateTask_Success() {
	title := "Updated Title"
	status := pb.TaskStatus_IN_PROGRESS
	priority := pb.TaskPriority_HIGH
	
	req := &pb.UpdateTaskRequest{
		Id:          suite.taskID,
		UserId:      suite.userID,
		Title:       title,
		Description: "Updated Description",
		Status:      status,
		Priority:    priority,
	}

	updatedTask := &model.Task{
		ID:          suite.taskID,
		UserID:      suite.userID,
		Title:       "Updated Title",
		Description: "Updated Description",
		Status:      model.StatusInProgress,
		Priority:    model.PriorityHigh,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Expectations
	suite.service.On("UpdateTask", suite.ctx, mock.AnythingOfType("*service.UpdateTaskRequest")).
		Return(updatedTask, nil).
		Once()

	// Test
	resp, err := suite.handler.UpdateTask(suite.ctx, req)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotNil(suite.T(), resp.Task)
	assert.Equal(suite.T(), "Updated Title", resp.Task.Title)
	assert.Equal(suite.T(), pb.TaskStatus_IN_PROGRESS, resp.Task.Status)
	assert.Equal(suite.T(), pb.TaskPriority_HIGH, resp.Task.Priority)
}

func (suite *TaskHandlerTestSuite) TestDeleteTask_Success() {
	req := &pb.DeleteTaskRequest{
		Id: suite.taskID,
	}

	// Expectations
	suite.service.On("DeleteTask", suite.ctx, suite.taskID).
		Return(nil).
		Once()

	// Test
	resp, err := suite.handler.DeleteTask(suite.ctx, req)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.True(suite.T(), resp.Success)
}

func (suite *TaskHandlerTestSuite) TestListTasks_Success() {
	req := &pb.ListTasksRequest{
		Page:            1,
		PageSize:        10,
		FilterByStatus:  "TODO",
		FilterByUserId:  suite.userID,
		SortBy:          "created_at",
		SortDesc:        true,
	}

	tasks := []*model.Task{
		{
			ID:          "task-1",
			UserID:      suite.userID,
			Title:       "Task 1",
			Description: "Description 1",
			Status:      model.StatusTodo,
			Priority:    model.PriorityMedium,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "task-2",
			UserID:      suite.userID,
			Title:       "Task 2",
			Description: "Description 2",
			Status:      model.StatusTodo,
			Priority:    model.PriorityHigh,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	
	const total int64 = 2

	// Expectations
	suite.service.On("ListTasks", suite.ctx, mock.AnythingOfType("*repository.TaskFilter"), 1, 10).
		Return(tasks, total, nil).
		Once()

	// Test
	resp, err := suite.handler.ListTasks(suite.ctx, req)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Len(suite.T(), resp.Tasks, 2)
	assert.Equal(suite.T(), int32(total), resp.Total)
	assert.Equal(suite.T(), int32(1), resp.Page)
	assert.Equal(suite.T(), int32(10), resp.PageSize)
	assert.Equal(suite.T(), "Task 1", resp.Tasks[0].Title)
	assert.Equal(suite.T(), "Task 2", resp.Tasks[1].Title)
}

func (suite *TaskHandlerTestSuite) TestListTasksByUser_Success() {
	req := &pb.ListTasksByUserRequest{
		UserId:           suite.userID,
		Page:             1,
		PageSize:         10,
		FilterByStatus:   "TODO",
		FilterByPriority: "MEDIUM",
		SortBy:           "created_at",
		SortDesc:         true,
	}

	tasks := []*model.Task{
		{
			ID:          "task-1",
			UserID:      suite.userID,
			Title:       "Task 1",
			Description: "Description 1",
			Status:      model.StatusTodo,
			Priority:    model.PriorityMedium,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	
	const total int64 = 1

	// Expectations
	suite.service.On("ListTasksByUser", suite.ctx, suite.userID, mock.AnythingOfType("*repository.TaskFilter"), 1, 10).
		Return(tasks, total, nil).
		Once()

	// Test
	resp, err := suite.handler.ListTasksByUser(suite.ctx, req)
	
	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Len(suite.T(), resp.Tasks, 1)
	assert.Equal(suite.T(), int32(total), resp.Total)
	assert.Equal(suite.T(), int32(1), resp.Page)
	assert.Equal(suite.T(), int32(10), resp.PageSize)
	assert.Equal(suite.T(), "Task 1", resp.Tasks[0].Title)
	assert.Equal(suite.T(), suite.userID, resp.Tasks[0].UserId)
}

func (suite *TaskHandlerTestSuite) TestHandlerErrorPropagation() {
	req := &pb.GetTaskRequest{
		Id: "non-existent-task",
	}

	// Expectations - service returns not found error
	suite.service.On("GetTask", suite.ctx, "non-existent-task").
		Return(nil, status.Error(codes.NotFound, "task not found")).
		Once()

	// Test
	resp, err := suite.handler.GetTask(suite.ctx, req)
	
	// Assertions
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	
	// Verify error is propagated correctly
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.NotFound, st.Code())
	assert.Equal(suite.T(), "task not found", st.Message())
}

func TestTaskHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(TaskHandlerTestSuite))
}