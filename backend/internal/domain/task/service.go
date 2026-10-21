package task

import (
	"errors"
	"time"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrInvalidStatus = errors.New("invalid task status")
)

type Service interface {
	CreateTask(t *Task) (*Task, error)
	GetTask(id uint) (*Task, error)
	ListTasks(filter Filter) ([]Task, int64, error)
	UpdateTask(id uint, updates map[string]interface{}) (*Task, error)
	CompleteTask(id uint) (*Task, error)
	DeleteTask(id uint) error
	AddComment(taskID uint, userID uint, commentText string) (*TaskComment, error)
	GetComments(taskID uint) ([]TaskComment, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateTask(t *Task) (*Task, error) {
	if t.Title == "" {
		return nil, errors.New("task title is required")
	}
	if t.Status == "" {
		t.Status = StatusTodo
	}
	if t.Priority == "" {
		t.Priority = PriorityMedium
	}
	if err := s.repo.Create(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *service) GetTask(id uint) (*Task, error) {
	t, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	return t, nil
}

func (s *service) ListTasks(filter Filter) ([]Task, int64, error) {
	return s.repo.List(filter)
}

func (s *service) UpdateTask(id uint, updates map[string]interface{}) (*Task, error) {
	t, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	if title, ok := updates["title"].(string); ok && title != "" {
		t.Title = title
	}
	if desc, ok := updates["description"].(string); ok {
		t.Description = desc
	}
	if status, ok := updates["status"].(string); ok && status != "" {
		t.Status = status
		if status == StatusCompleted && t.CompletedAt == nil {
			now := time.Now()
			t.CompletedAt = &now
		}
	}
	if priority, ok := updates["priority"].(string); ok && priority != "" {
		t.Priority = priority
	}
	if assigneeID, ok := updates["assignee_id"].(float64); ok {
		uid := uint(assigneeID)
		t.AssigneeID = &uid
	}

	if err := s.repo.Update(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *service) CompleteTask(id uint) (*Task, error) {
	t, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	t.Status = StatusCompleted
	now := time.Now()
	t.CompletedAt = &now
	if err := s.repo.Update(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *service) DeleteTask(id uint) error {
	return s.repo.Delete(id)
}

func (s *service) AddComment(taskID uint, userID uint, commentText string) (*TaskComment, error) {
	if commentText == "" {
		return nil, errors.New("comment text cannot be empty")
	}
	comment := &TaskComment{
		TaskID:  taskID,
		UserID:  userID,
		Comment: commentText,
	}
	if err := s.repo.AddComment(comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *service) GetComments(taskID uint) ([]TaskComment, error) {
	return s.repo.GetComments(taskID)
}
