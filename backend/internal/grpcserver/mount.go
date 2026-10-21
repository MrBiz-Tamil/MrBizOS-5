package grpcserver

import (
	"drayya-core/gen/healthcare/v1/healthcarev1connect"
	"drayya-core/gen/iam/v1/iamv1connect"
	"drayya-core/gen/realestate/v1/realestatev1connect"
	"drayya-core/gen/travel/v1/travelv1connect"
	"drayya-core/gen/workflow/v1/workflowv1connect"

	"drayya-core/internal/grpcservices/healthcare"
	"drayya-core/internal/grpcservices/iam"
	"drayya-core/internal/grpcservices/realestate"
	"drayya-core/internal/grpcservices/travel"
	"drayya-core/internal/grpcservices/workflow"
)

// RegisterAllServices attaches all canonical domain services to the Connect server mux
func (s *Server) RegisterAllServices() {
	opt := s.Interceptor()

	// 1. IAM & Multi-Unit Service
	iamSvc := iam.NewServer(s.db)
	iamPath, iamHandler := iamv1connect.NewIAMServiceHandler(iamSvc, opt)
	s.mux.Handle(iamPath, iamHandler)

	// 2. Healthcare & Telemedicine Service
	healthSvc := healthcare.NewServer(s.db)
	healthPath, healthHandler := healthcarev1connect.NewHealthcareServiceHandler(healthSvc, opt)
	s.mux.Handle(healthPath, healthHandler)

	// 3. Travel, Tours & Visa Service
	travelSvc := travel.NewServer(s.db)
	travelPath, travelHandler := travelv1connect.NewTravelServiceHandler(travelSvc, opt)
	s.mux.Handle(travelPath, travelHandler)

	// 4. Real Estate & Deal Service
	realestateSvc := realestate.NewServer(s.db)
	rePath, reHandler := realestatev1connect.NewRealEstateServiceHandler(realestateSvc, opt)
	s.mux.Handle(rePath, reHandler)

	// 5. Workflow, Tasks & Approval Queue Service
	wfSvc := workflow.NewServer(s.db)
	wfPath, wfHandler := workflowv1connect.NewWorkflowServiceHandler(wfSvc, opt)
	s.mux.Handle(wfPath, wfHandler)
}
