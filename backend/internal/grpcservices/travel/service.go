package travel

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"
	travelv1 "drayya-core/gen/travel/v1"
	"drayya-core/gen/travel/v1/travelv1connect"
	"drayya-core/internal/authctx"
	"drayya-core/internal/database"
	traveldomain "drayya-core/internal/domain/travel"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

// orgID returns the authenticated caller's organization id from the request
// context. It fails closed: an unauthenticated/unscoped context yields 0, which
// the org scope treats as matching no rows.
func orgID(ctx context.Context) uint {
	return database.OrgIDFromTenant(authctx.TenantID(ctx))
}

func (s *Server) ListTourPackages(ctx context.Context, req *connect.Request[travelv1.ListTourPackagesRequest]) (*connect.Response[travelv1.ListTourPackagesResponse], error) {
	var pkgs []*travelv1.TourPackageProto

	// Fail closed: with no authorized tenant on the context there is no org to
	// scope to, so we return nothing rather than leaking another tenant's data.
	org := orgID(ctx)
	tenantStr := authctx.TenantID(ctx)

	if s.db != nil && org != 0 {
		var dbPkgs []traveldomain.TourPackage
		// Reads are restricted to the caller's own organization.
		s.db.Scopes(database.OrgScope(org)).Find(&dbPkgs)
		if len(dbPkgs) == 0 {
			// Seed baseline packages for THIS tenant only (demo convenience).
			p1 := traveldomain.TourPackage{
				OrganizationID: org,
				BusinessUnitID: 1,
				Title:          "Golden Triangle Deluxe Tour",
				Slug:           fmt.Sprintf("golden-triangle-deluxe-%d", org),
				Days:           6,
				Nights:         5,
				Price:          24999.00,
				Highlights:     "Delhi, Agra, Jaipur",
				IsActive:       true,
			}
			p2 := traveldomain.TourPackage{
				OrganizationID: org,
				BusinessUnitID: 1,
				Title:          "Dubai Sky & Desert Safari",
				Slug:           fmt.Sprintf("dubai-sky-desert-safari-%d", org),
				Days:           5,
				Nights:         4,
				Price:          45000.00,
				Highlights:     "Dubai, UAE",
				IsActive:       true,
			}
			s.db.Create(&p1)
			s.db.Create(&p2)
			dbPkgs = []traveldomain.TourPackage{p1, p2}
		}

		for _, p := range dbPkgs {
			pkgs = append(pkgs, &travelv1.TourPackageProto{
				Id:            fmt.Sprintf("tour-%d", p.ID),
				TenantId:      tenantStr,
				Title:         p.Title,
				Destination:   p.Highlights,
				DurationDays:  int32(p.Days),
				Price:         fmt.Sprintf("%.2f", p.Price),
				ItineraryJson: `[{"day":1,"title":"Arrival & Check-in"},{"day":2,"title":"Guided Excursion"}]`,
				IsActive:      p.IsActive,
			})
		}
	}

	return connect.NewResponse(&travelv1.ListTourPackagesResponse{Packages: pkgs}), nil
}

func (s *Server) GetTourPackage(ctx context.Context, req *connect.Request[travelv1.GetTourPackageRequest]) (*connect.Response[travelv1.GetTourPackageResponse], error) {
	org := orgID(ctx)
	if s.db == nil || org == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("tenant context required"))
	}
	var pkg traveldomain.TourPackage
	// Scoped lookup: a package is only visible to its owning organization, so a
	// package id from another tenant resolves to not-found rather than leaking.
	if err := s.db.Scopes(database.OrgScope(org)).First(&pkg, "slug = ? OR id = ?", req.Msg.Id, req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tour package not found: %s", req.Msg.Id))
	}
	return connect.NewResponse(&travelv1.GetTourPackageResponse{
		Package: &travelv1.TourPackageProto{
			Id:           fmt.Sprintf("tour-%d", pkg.ID),
			TenantId:     authctx.TenantID(ctx),
			Title:        pkg.Title,
			Destination:  pkg.Highlights,
			DurationDays: int32(pkg.Days),
			Price:        fmt.Sprintf("%.2f", pkg.Price),
			IsActive:     pkg.IsActive,
		},
	}), nil
}

func (s *Server) CreateTourPackage(ctx context.Context, req *connect.Request[travelv1.CreateTourPackageRequest]) (*connect.Response[travelv1.CreateTourPackageResponse], error) {
	org := orgID(ctx)
	if org == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("tenant context required"))
	}
	priceVal, _ := strconv.ParseFloat(req.Msg.Price, 64)
	newPkg := traveldomain.TourPackage{
		OrganizationID: org,
		BusinessUnitID: 1,
		Title:          req.Msg.Title,
		Slug:           fmt.Sprintf("tour-%d", time.Now().Unix()),
		Days:           int(req.Msg.DurationDays),
		Price:          priceVal,
		Highlights:     req.Msg.Destination,
		IsActive:       true,
	}

	if s.db != nil {
		_ = s.db.Create(&newPkg).Error
	}

	idStr := fmt.Sprintf("tour-%d", newPkg.ID)
	if newPkg.ID == 0 {
		idStr = fmt.Sprintf("tour-%d", time.Now().Unix())
	}

	pkgProto := &travelv1.TourPackageProto{
		Id:            idStr,
		TenantId:      authctx.TenantID(ctx),
		Title:         req.Msg.Title,
		Destination:   req.Msg.Destination,
		DurationDays:  req.Msg.DurationDays,
		Price:         req.Msg.Price,
		ItineraryJson: req.Msg.ItineraryJson,
		IsActive:      true,
	}
	return connect.NewResponse(&travelv1.CreateTourPackageResponse{Package: pkgProto}), nil
}

func (s *Server) ListVisaApplications(ctx context.Context, req *connect.Request[travelv1.ListVisaApplicationsRequest]) (*connect.Response[travelv1.ListVisaApplicationsResponse], error) {
	var apps []*travelv1.VisaApplicationProto

	org := orgID(ctx)
	tenantStr := authctx.TenantID(ctx)

	if s.db != nil && org != 0 {
		var dbVisas []traveldomain.VisaApplication
		s.db.Scopes(database.OrgScope(org)).Find(&dbVisas)
		if len(dbVisas) == 0 {
			v1 := traveldomain.VisaApplication{
				OrganizationID: org,
				BusinessUnitID: 1,
				ApplicantID:    1,
				PassportNumber: "Z8291048",
				CountryCode:    "United Kingdom",
				VisaType:       "Tourist Standard",
				Status:         "UNDER_REVIEW",
				Notes:          "Aakash Sharma",
			}
			s.db.Create(&v1)
			dbVisas = []traveldomain.VisaApplication{v1}
		}

		for _, v := range dbVisas {
			applicantName := v.Notes
			if applicantName == "" {
				applicantName = "Aakash Sharma"
			}
			apps = append(apps, &travelv1.VisaApplicationProto{
				Id:             fmt.Sprintf("visa-%d", v.ID),
				TenantId:       tenantStr,
				ApplicantName:  applicantName,
				PassportNumber: v.PassportNumber,
				Country:        v.CountryCode,
				VisaType:       v.VisaType,
				Status:         v.Status,
				ChecklistJson:  `{"passport":true,"bank_statement":true,"invitation_letter":false}`,
				SubmittedAt:    v.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	return connect.NewResponse(&travelv1.ListVisaApplicationsResponse{Applications: apps}), nil
}

func (s *Server) SubmitVisaApplication(ctx context.Context, req *connect.Request[travelv1.SubmitVisaApplicationRequest]) (*connect.Response[travelv1.SubmitVisaApplicationResponse], error) {
	org := orgID(ctx)
	if org == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("tenant context required"))
	}
	newVisa := traveldomain.VisaApplication{
		OrganizationID: org,
		BusinessUnitID: 1,
		ApplicantID:    1,
		CountryCode:    req.Msg.Country,
		VisaType:       req.Msg.VisaType,
		PassportNumber: req.Msg.PassportNumber,
		Status:         "SUBMITTED",
		Notes:          req.Msg.ApplicantName,
	}

	if s.db != nil {
		_ = s.db.Create(&newVisa).Error
	}

	idStr := fmt.Sprintf("visa-%d", newVisa.ID)
	if newVisa.ID == 0 {
		idStr = fmt.Sprintf("visa-%d", time.Now().Unix())
	}

	app := &travelv1.VisaApplicationProto{
		Id:             idStr,
		TenantId:       authctx.TenantID(ctx),
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
	if s.db != nil {
		// Scope the update to the caller's organization so one tenant cannot
		// change another tenant's visa application by id.
		s.db.Model(&traveldomain.VisaApplication{}).
			Where("id = ? AND organization_id = ?", req.Msg.Id, orgID(ctx)).
			Update("status", req.Msg.Status)
	}

	return connect.NewResponse(&travelv1.UpdateVisaStatusResponse{
		Application: &travelv1.VisaApplicationProto{
			Id:     req.Msg.Id,
			Status: req.Msg.Status,
		},
	}), nil
}

// Compile-time check
var _ travelv1connect.TravelServiceHandler = (*Server)(nil)
