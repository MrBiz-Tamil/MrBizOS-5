package workflow

import (
	"context"
	"fmt"
	"time"

	"strconv"

	"connectrpc.com/connect"
	workflowv1 "drayya-core/gen/workflow/v1"
	"drayya-core/gen/workflow/v1/workflowv1connect"
	"drayya-core/internal/authctx"
	"drayya-core/internal/database"
	taskdomain "drayya-core/internal/domain/task"
	"gorm.io/gorm"
)

// orgID returns the authenticated caller's organization id from the request
// context. It fails closed: an unauthenticated/unscoped context yields 0, which
// the org scope treats as matching no rows.
func orgID(ctx context.Context) uint {
	return database.OrgIDFromTenant(authctx.TenantID(ctx))
}

// assigneeStr renders a task's assignee id (nil-safe), falling back to the owner.
func assigneeStr(t taskdomain.Task) string {
	if t.AssigneeID != nil {
		return fmt.Sprintf("%d", *t.AssigneeID)
	}
	return fmt.Sprintf("%d", t.OwnerID)
}

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ListTasks(ctx context.Context, req *connect.Request[workflowv1.ListTasksRequest]) (*connect.Response[workflowv1.ListTasksResponse], error) {
	var tasks []*workflowv1.TaskProto

	// Fail closed: with no authorized tenant on the context there is no org to
	// scope to, so we return nothing rather than leaking another tenant's tasks.
	org := orgID(ctx)
	tenantStr := authctx.TenantID(ctx)

	if s.db != nil && org != 0 {
		var dbTasks []taskdomain.Task
		// Reads are restricted to the caller's own organization.
		s.db.Scopes(database.OrgScope(org)).Find(&dbTasks)
		if len(dbTasks) == 0 {
			// Seed baseline tasks for THIS tenant only (demo convenience; scoped
			// so it never seeds into or reads from another tenant).
			t1 := taskdomain.Task{
				OrganizationID: org,
				Title:          "Verify Doctor Medical Council License",
				Description:    "Validate registration details with state medical council",
				Priority:       "HIGH",
				Status:         "PENDING",
				OwnerID:        1,
			}
			t2 := taskdomain.Task{
				OrganizationID: org,
				Title:          "Process Visa Application Documents",
				Description:    "Check passport scan clarity and bank statements",
				Priority:       "MEDIUM",
				Status:         "IN_PROGRESS",
				OwnerID:        1,
			}
			s.db.Create(&t1)
			s.db.Create(&t2)
			dbTasks = []taskdomain.Task{t1, t2}
		}

		for _, t := range dbTasks {
			tasks = append(tasks, &workflowv1.TaskProto{
				Id:             fmt.Sprintf("task-%d", t.ID),
				TenantId:       tenantStr,
				BusinessUnitId: req.Msg.BusinessUnitId,
				Title:          t.Title,
				Description:    t.Description,
				Priority:       t.Priority,
				Status:         t.Status,
				AssigneeId:     assigneeStr(t),
				DueDate:        time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			})
		}
	}

	return connect.NewResponse(&workflowv1.ListTasksResponse{Tasks: tasks}), nil
}

func (s *Server) CreateTask(ctx context.Context, req *connect.Request[workflowv1.CreateTaskRequest]) (*connect.Response[workflowv1.CreateTaskResponse], error) {
	org := orgID(ctx)
	if org == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("tenant context required"))
	}
	var ownerID uint = 1
	if uid, err := strconv.ParseUint(authctx.UserID(ctx), 10, 64); err == nil && uid != 0 {
		ownerID = uint(uid)
	}

	newTask := taskdomain.Task{
		OrganizationID: org,
		Title:          req.Msg.Title,
		Description:    req.Msg.Description,
		Priority:       req.Msg.Priority,
		Status:         "PENDING",
		OwnerID:        ownerID,
	}

	if s.db != nil {
		_ = s.db.Create(&newTask).Error
	}

	idStr := fmt.Sprintf("task-%d", newTask.ID)
	if newTask.ID == 0 {
		idStr = fmt.Sprintf("task-%d", time.Now().Unix())
	}

	task := &workflowv1.TaskProto{
		Id:             idStr,
		TenantId:       authctx.TenantID(ctx),
		BusinessUnitId: req.Msg.BusinessUnitId,
		Title:          req.Msg.Title,
		Description:    req.Msg.Description,
		Priority:       req.Msg.Priority,
		Status:         "PENDING",
		AssigneeId:     req.Msg.AssigneeId,
		DueDate:        req.Msg.DueDate,
	}
	return connect.NewResponse(&workflowv1.CreateTaskResponse{Task: task}), nil
}

func (s *Server) UpdateTaskStatus(ctx context.Context, req *connect.Request[workflowv1.UpdateTaskStatusRequest]) (*connect.Response[workflowv1.UpdateTaskStatusResponse], error) {
	if s.db != nil {
		// Scope the update to the caller's organization so one tenant cannot
		// mutate another tenant's task by id.
		s.db.Model(&taskdomain.Task{}).
			Where("id = ? AND organization_id = ?", req.Msg.Id, orgID(ctx)).
			Update("status", req.Msg.Status)
	}

	return connect.NewResponse(&workflowv1.UpdateTaskStatusResponse{
		Task: &workflowv1.TaskProto{
			Id:     req.Msg.Id,
			Status: req.Msg.Status,
		},
	}), nil
}

func (s *Server) ListApprovalQueue(ctx context.Context, req *connect.Request[workflowv1.ListApprovalQueueRequest]) (*connect.Response[workflowv1.ListApprovalQueueResponse], error) {
	items := []*workflowv1.ApprovalItemProto{
		{
			Id:             "appr-1",
			TenantId:       "tenant-1",
			BusinessUnitId: "realestate",
			Source:         "AI_AGENT",
			Title:          "Autonomous Price Offer on Emerald Heights",
			ActionType:     "OFFER_SUBMISSION",
			PayloadJson:    `{"propertyId":"prop-1","offeredAmount":"17500000","counterpart":"Dev Developers"}`,
			RequestedBy:    "Skyline Negotiation Agent",
			Status:         "PENDING",
			CreatedAt:      time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
		{
			Id:             "appr-2",
			TenantId:       "tenant-1",
			BusinessUnitId: "travel",
			Source:         "LOGISTICS",
			Title:          "Confirm Dubai Hotel Bulk Allocation",
			ActionType:     "VENDOR_BOOKING",
			PayloadJson:    `{"vendor":"Marriott Dubai","totalRooms":15,"totalAmount":"240000"}`,
			RequestedBy:    "Travel Operations",
			Status:         "PENDING",
			CreatedAt:      time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
		},
	}
	return connect.NewResponse(&workflowv1.ListApprovalQueueResponse{Items: items}), nil
}

func (s *Server) ApproveAction(ctx context.Context, req *connect.Request[workflowv1.ApproveActionRequest]) (*connect.Response[workflowv1.ApproveActionResponse], error) {
	return connect.NewResponse(&workflowv1.ApproveActionResponse{
		Success: true,
		Message: fmt.Sprintf("Action %s approved successfully", req.Msg.Id),
	}), nil
}

func (s *Server) RejectAction(ctx context.Context, req *connect.Request[workflowv1.RejectActionRequest]) (*connect.Response[workflowv1.RejectActionResponse], error) {
	return connect.NewResponse(&workflowv1.RejectActionResponse{
		Success: true,
		Message: fmt.Sprintf("Action %s rejected", req.Msg.Id),
	}), nil
}

// Compile-time check
var _ workflowv1connect.WorkflowServiceHandler = (*Server)(nil)
