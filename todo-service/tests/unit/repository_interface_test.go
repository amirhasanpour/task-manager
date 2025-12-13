package tests

import (
	"context"
	"testing"

	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
)

// TestRepositoryInterface is a compile-time test to ensure the interface is implemented
// We need to create a concrete type that implements the interface
type testRepositoryImpl struct{}

func (t *testRepositoryImpl) Create(ctx context.Context, task *model.Task) (*model.Task, error) {
	return nil, nil
}

func (t *testRepositoryImpl) FindByID(ctx context.Context, id string) (*model.Task, error) {
	return nil, nil
}

func (t *testRepositoryImpl) FindByIDAndUser(ctx context.Context, id, userID string) (*model.Task, error) {
	return nil, nil
}

func (t *testRepositoryImpl) Update(ctx context.Context, task *model.Task) (*model.Task, error) {
	return nil, nil
}

func (t *testRepositoryImpl) Delete(ctx context.Context, id string) error {
	return nil
}

func (t *testRepositoryImpl) DeleteByUser(ctx context.Context, id, userID string) error {
	return nil
}

func (t *testRepositoryImpl) List(ctx context.Context, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	return nil, 0, nil
}

func (t *testRepositoryImpl) ListByUser(ctx context.Context, userID string, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	return nil, 0, nil
}

func TestRepositoryInterface(t *testing.T) {
	// Create an instance of our test implementation
	var repo repository.TaskRepository = &testRepositoryImpl{}
	
	// This will fail at compile time if testRepositoryImpl doesn't implement all methods
	_ = repo
	
	t.Log("Repository interface check passed - all methods are implemented")
}

// TestTaskFilterDefaults tests basic filter functionality
func TestTaskFilterDefaults(t *testing.T) {
	filter := &repository.TaskFilter{}
	
	if filter.Status != nil {
		t.Errorf("Expected Status to be nil, got %v", *filter.Status)
	}
	if filter.Priority != nil {
		t.Errorf("Expected Priority to be nil, got %v", *filter.Priority)
	}
	if filter.UserID != nil {
		t.Errorf("Expected UserID to be nil, got %v", *filter.UserID)
	}
	if filter.SortBy != "" {
		t.Errorf("Expected SortBy to be empty, got %s", filter.SortBy)
	}
	if filter.SortDesc {
		t.Error("Expected SortDesc to be false")
	}
}

// TestModelEnums tests enum values
func TestModelEnums(t *testing.T) {
	// Test TaskStatus enum
	statusTests := []struct {
		status   model.TaskStatus
		expected string
	}{
		{model.StatusTodo, "todo"},
		{model.StatusInProgress, "in_progress"},
		{model.StatusDone, "done"},
		{model.StatusArchived, "archived"},
	}
	
	for _, tt := range statusTests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected status %s, got %s", tt.expected, tt.status)
		}
	}
	
	// Test TaskPriority enum
	priorityTests := []struct {
		priority model.TaskPriority
		expected string
	}{
		{model.PriorityLow, "low"},
		{model.PriorityMedium, "medium"},
		{model.PriorityHigh, "high"},
		{model.PriorityUrgent, "urgent"},
	}
	
	for _, tt := range priorityTests {
		if string(tt.priority) != tt.expected {
			t.Errorf("Expected priority %s, got %s", tt.expected, tt.priority)
		}
	}
}