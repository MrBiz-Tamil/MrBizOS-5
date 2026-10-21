package workflow

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	workflowv1 "drayya-core/gen/workflow/v1"
	"drayya-core/gen/workflow/v1/workflowv1connect"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ListTasks(ctx context.Context, req *connect.Request[workflowv1.ListTasksRequest]) (*connect.Response[workflowv1.ListTasksResponse], error) {
	tasks := []*workflowv1.TaskProto{
		{
			Id:             "task-1",
			TenantId:       "tenant-1",
			BusinessUnitId: req.Msg.BusinessUnitId,
			Title:          "Verify Doctor Medical Council License",
			Description:    "Validate registration details with state medical council",
			Priority:       "HIGH",
			Status:         "PENDING",
			AssigneeId:     "user-1",
			DueDate:        time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		},
		{
			Id:             "task-2",
			TenantId:       "tenant-1",
			BusinessUnitId: req.Msg.BusinessUnitId,
			Title:          "Process Visa Application Documents",
			Description:    "Check passport scan clarity and bank statements",
			Priority:       "MEDIUM",
			Status:         "IN_PROGRESS",
			AssigneeId:     "user-1",
			DueDate:        time.Now().Add(48 * time.Hour).Format(time.RFC3339),
		},
	}
	return connect.NewResponse(&workflowv1.ListTasksResponse{Tasks: tasks}), nil
}

func (s *Server) CreateTask(ctx context.Context, req *connect.Request[workflowv1.CreateTaskRequest]) (*connect.Response[workflowv1.CreateTaskResponse], error) {
	task := &workflowv1.TaskProto{
		Id:             fmt.Sprintf("task-%d", time.Now().Unix()),
		TenantId:       "tenant-1",
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
