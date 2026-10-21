package task

import (
	"gorm.io/gorm"
)

type Repository interface {
	Create(task *Task) error
	GetByID(id uint) (*Task, error)
	List(filter Filter) ([]Task, int64, error)
	Update(task *Task) error
	Delete(id uint) error
	AddComment(comment *TaskComment) error
	GetComments(taskID uint) ([]TaskComment, error)
}

type Filter struct {
	OrganizationID uint
	BusinessUnitID uint
	AssigneeID     *uint
	OwnerID        *uint
	CustomerID     *uint
	Status         string
	Priority       string
	Search         string
	Page           int
	Limit          int
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(task *Task) error {
	return r.db.Create(task).Error
}

func (r *repository) GetByID(id uint) (*Task, error) {
	var task Task
	if err := r.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *repository) List(filter Filter) ([]Task, int64, error) {
	var tasks []Task
	var total int64

	q := r.db.Model(&Task{})

	if filter.OrganizationID > 0 {
		q = q.Where("organization_id = ?", filter.OrganizationID)
	}
	if filter.BusinessUnitID > 0 {
		q = q.Where("business_unit_id = ?", filter.BusinessUnitID)
	}
	if filter.AssigneeID != nil {
		q = q.Where("assignee_id = ?", *filter.AssigneeID)
	}
	if filter.OwnerID != nil {
		q = q.Where("owner_id = ?", *filter.OwnerID)
	}
	if filter.CustomerID != nil {
		q = q.Where("customer_id = ?", *filter.CustomerID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Priority != "" {
		q = q.Where("priority = ?", filter.Priority)
	}
	if filter.Search != "" {
		s := "%" + filter.Search + "%"
		q = q.Where("title LIKE ? OR description LIKE ? OR tags LIKE ?", s, s, s)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	if err := q.Order("due_date ASC, priority DESC, created_at DESC").Offset(offset).Limit(limit).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (r *repository) Update(task *Task) error {
	return r.db.Save(task).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&Task{}, id).Error
}

func (r *repository) AddComment(comment *TaskComment) error {
	return r.db.Create(comment).Error
}

func (r *repository) GetComments(taskID uint) ([]TaskComment, error) {
	var comments []TaskComment
	err := r.db.Where("task_id = ?", taskID).Order("created_at ASC").Find(&comments).Error
	return comments, err
}
