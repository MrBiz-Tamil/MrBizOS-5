package grpcserver

import (
	"drayya-core/gen/banking/v1/bankingv1connect"
	"drayya-core/gen/healthcare/v1/healthcarev1connect"
	"drayya-core/gen/hrms/v1/hrmsv1connect"
	"drayya-core/gen/iam/v1/iamv1connect"
	"drayya-core/gen/journey/v1/journeyv1connect"
	"drayya-core/gen/omnichannel/v1/omnichannelv1connect"
	"drayya-core/gen/procurement/v1/procurementv1connect"
	"drayya-core/gen/project/v1/projectv1connect"
	"drayya-core/gen/realestate/v1/realestatev1connect"
	"drayya-core/gen/travel/v1/travelv1connect"
	"drayya-core/gen/workflow/v1/workflowv1connect"

	"drayya-core/internal/grpcservices/banking"
	"drayya-core/internal/grpcservices/healthcare"
	"drayya-core/internal/grpcservices/hrms"
	"drayya-core/internal/grpcservices/iam"
	"drayya-core/internal/grpcservices/journey"
	"drayya-core/internal/grpcservices/omnichannel"
	"drayya-core/internal/grpcservices/procurement"
	"drayya-core/internal/grpcservices/project"
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

	// 6. Procurement & 3-Way Matching Service
	procSvc := procurement.NewServer(s.db)
	procPath, procHandler := procurementv1connect.NewProcurementServiceHandler(procSvc, opt)
	s.mux.Handle(procPath, procHandler)

	// 7. Banking & Automated Reconciliation Service
	bankSvc := banking.NewServer(s.db)
	bankPath, bankHandler := bankingv1connect.NewBankingServiceHandler(bankSvc, opt)
	s.mux.Handle(bankPath, bankHandler)

	// 8. HRMS & Automated Payroll Service
	hrmsSvc := hrms.NewServer(s.db)
	hrmsPath, hrmsHandler := hrmsv1connect.NewHRMSServiceHandler(hrmsSvc, opt)
	s.mux.Handle(hrmsPath, hrmsHandler)

	// 9. Project & UTMS Task Management Service
	projSvc := project.NewService(s.db)
	projPath, projHandler := projectv1connect.NewProjectServiceHandler(projSvc, opt)
	s.mux.Handle(projPath, projHandler)

	// 10. Journey Management Service
	journeySvc := journey.NewService(s.db)
	journeyPath, journeyHandler := journeyv1connect.NewJourneyServiceHandler(journeySvc, opt)
	s.mux.Handle(journeyPath, journeyHandler)

	// 11. Omnichannel Social & Communication Service (WhatsApp, Instagram, FB, Email)
	omniSvc := omnichannel.NewService(s.db)
	omniPath, omniHandler := omnichannelv1connect.NewOmnichannelServiceHandler(omniSvc, opt)
	s.mux.Handle(omniPath, omniHandler)
}
