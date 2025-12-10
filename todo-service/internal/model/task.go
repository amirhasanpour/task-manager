package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskStatus string
type TaskPriority string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
	StatusArchived   TaskStatus = "archived"
)

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
	PriorityUrgent TaskPriority = "urgent"
)

type Task struct {
	ID          string       `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID      string       `gorm:"type:uuid;not null;index:idx_user_id" json:"user_id"`
	Title       string       `gorm:"type:varchar(255);not null" json:"title"`
	Description string       `gorm:"type:text" json:"description"`
	Status      TaskStatus   `gorm:"type:varchar(20);not null;default:'todo';index:idx_status" json:"status"`
	Priority    TaskPriority `gorm:"type:varchar(20);not null;default:'medium';index:idx_priority" json:"priority"`
	DueDate     *time.Time   `gorm:"index:idx_due_date" json:"due_date"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

func (t *Task) ToProtoStatus() string {
	switch t.Status {
	case StatusTodo:
		return "TODO"
	case StatusInProgress:
		return "IN_PROGRESS"
	case StatusDone:
		return "DONE"
	case StatusArchived:
		return "ARCHIVED"
	default:
		return "TODO"
	}
}

func (t *Task) ToProtoPriority() string {
	switch t.Priority {
	case PriorityLow:
		return "LOW"
	case PriorityMedium:
		return "MEDIUM"
	case PriorityHigh:
		return "HIGH"
	case PriorityUrgent:
		return "URGENT"
	default:
		return "MEDIUM"
	}
}

func (t *Task) FromProtoStatus(status string) TaskStatus {
	switch status {
	case "TODO":
		return StatusTodo
	case "IN_PROGRESS":
		return StatusInProgress
	case "DONE":
		return StatusDone
	case "ARCHIVED":
		return StatusArchived
	default:
		return StatusTodo
	}
}

func (t *Task) FromProtoPriority(priority string) TaskPriority {
	switch priority {
	case "LOW":
		return PriorityLow
	case "MEDIUM":
		return PriorityMedium
	case "HIGH":
		return PriorityHigh
	case "URGENT":
		return PriorityUrgent
	default:
		return PriorityMedium
	}
}