package travel

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	travelv1 "drayya-core/gen/travel/v1"
	"drayya-core/gen/travel/v1/travelv1connect"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ListTourPackages(ctx context.Context, req *connect.Request[travelv1.ListTourPackagesRequest]) (*connect.Response[travelv1.ListTourPackagesResponse], error) {
	pkgs := []*travelv1.TourPackageProto{
		{
			Id:            "tour-1",
			TenantId:      "tenant-1",
			Title:         "Golden Triangle Deluxe Tour",
			Destination:   "Delhi, Agra, Jaipur",
			DurationDays:  6,
			Price:         "24999.00",
			ItineraryJson: `[{"day":1,"title":"Arrival Delhi"},{"day":2,"title":"Agra Taj Mahal"},{"day":3,"title":"Jaipur City Palace"}]`,
			IsActive:      true,
		},
		{
			Id:            "tour-2",
			TenantId:      "tenant-1",
			Title:         "Dubai Sky & Desert Safari",
			Destination:   "Dubai, UAE",
			DurationDays:  5,
			Price:         "45000.00",
			ItineraryJson: `[{"day":1,"title":"Burj Khalifa"},{"day":2,"title":"Dune Bashing"}]`,
			IsActive:      true,
		},
	}
	return connect.NewResponse(&travelv1.ListTourPackagesResponse{Packages: pkgs}), nil
}

func (s *Server) GetTourPackage(ctx context.Context, req *connect.Request[travelv1.GetTourPackageRequest]) (*connect.Response[travelv1.GetTourPackageResponse], error) {
	return connect.NewResponse(&travelv1.GetTourPackageResponse{
		Package: &travelv1.TourPackageProto{
			Id:           req.Msg.Id,
			TenantId:     "tenant-1",
			Title:        "Golden Triangle Deluxe Tour",
			Destination:  "Delhi, Agra, Jaipur",
			DurationDays: 6,
			Price:        "24999.00",
			IsActive:     true,
		},
	}), nil
}

func (s *Server) CreateTourPackage(ctx context.Context, req *connect.Request[travelv1.CreateTourPackageRequest]) (*connect.Response[travelv1.CreateTourPackageResponse], error) {
	pkg := &travelv1.TourPackageProto{
		Id:            fmt.Sprintf("tour-%d", time.Now().Unix()),
		TenantId:      "tenant-1",
		Title:         req.Msg.Title,
		Destination:   req.Msg.Destination,
		DurationDays:  req.Msg.DurationDays,
		Price:         req.Msg.Price,
		ItineraryJson: req.Msg.ItineraryJson,
		IsActive:      true,
	}
	return connect.NewResponse(&travelv1.CreateTourPackageResponse{Package: pkg}), nil
}

func (s *Server) ListVisaApplications(ctx context.Context, req *connect.Request[travelv1.ListVisaApplicationsRequest]) (*connect.Response[travelv1.ListVisaApplicationsResponse], error) {
	apps := []*travelv1.VisaApplicationProto{
		{
			Id:              "visa-1",
			TenantId:        "tenant-1",
			ApplicantName:   "Aakash Sharma",
			PassportNumber:  "Z8291048",
			Country:         "United Kingdom",
			VisaType:        "Tourist Standard",
			Status:          "UNDER_REVIEW",
			ChecklistJson:   `{"passport":true,"bank_statement":true,"invitation_letter":false}`,
			SubmittedAt:     time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		},
	}
	return connect.NewResponse(&travelv1.ListVisaApplicationsResponse{Applications: apps}), nil
}

func (s *Server) SubmitVisaApplication(ctx context.Context, req *connect.Request[travelv1.SubmitVisaApplicationRequest]) (*connect.Response[travelv1.SubmitVisaApplicationResponse], error) {
	app := &travelv1.VisaApplicationProto{
		Id:             fmt.Sprintf("visa-%d", time.Now().Unix()),
		TenantId:       "tenant-1",
		ApplicantName:  req.Msg.ApplicantName,
		PassportNumber: req.Msg.PassportNumber,
		Country:        req.Msg.Country,
		VisaType:       req.Msg.VisaType,
		Status:         "SUBMITTED",
		ChecklistJson:  req.Msg.ChecklistJson,
		SubmittedAt:    time.Now().Format(time.RFC3339),
	}
	return connect.NewResponse(&travelv1.SubmitVisaApplicationResponse{Application: app}), nil
}

func (s *Server) UpdateVisaStatus(ctx context.Context, req *connect.Request[travelv1.UpdateVisaStatusRequest]) (*connect.Response[travelv1.UpdateVisaStatusResponse], error) {
	return connect.NewResponse(&travelv1.UpdateVisaStatusResponse{
		Application: &travelv1.VisaApplicationProto{
			Id:     req.Msg.Id,
			Status: req.Msg.Status,
		},
	}), nil
}

// Compile-time check
var _ travelv1connect.TravelServiceHandler = (*Server)(nil)
