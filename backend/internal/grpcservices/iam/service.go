package iam

import (
	"context"

	"connectrpc.com/connect"
	iamv1 "drayya-core/gen/iam/v1"
	"drayya-core/gen/iam/v1/iamv1connect"
	"drayya-core/internal/tokens"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) Login(ctx context.Context, req *connect.Request[iamv1.LoginRequest]) (*connect.Response[iamv1.LoginResponse], error) {
	// For testing/prototype, generate a valid JWT token
	token, err := tokens.CreateAccessToken("user-1", "admin", nil, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := connect.NewResponse(&iamv1.LoginResponse{
		User: &iamv1.UserProto{
			Id:                   "user-1",
			TenantId:             "tenant-1",
			Email:                req.Msg.Email,
			FullName:             "Executive Administrator",
			Role:                 "admin",
			IsSuperAdmin:         true,
			IsActive:             true,
			CurrentBusinessUnitId: "healthcare",
			AllowedBusinessUnits: []string{"healthcare", "travel", "realestate", "ecommerce", "consulting"},
		},
		Tenant: &iamv1.TenantProto{
			Id:                 "tenant-1",
			Name:               "Global Business Group",
			LegalName:          "Global Business Group Ltd",
			Currency:           "INR",
			SubscriptionPlan:   "enterprise",
			SubscriptionStatus: "active",
			IsActive:           true,
		},
		Token: token,
	})
	return res, nil
}

func (s *Server) Logout(ctx context.Context, req *connect.Request[iamv1.LogoutRequest]) (*connect.Response[iamv1.LogoutResponse], error) {
	return connect.NewResponse(&iamv1.LogoutResponse{Message: "Logged out successfully"}), nil
}

func (s *Server) GetMe(ctx context.Context, req *connect.Request[iamv1.GetMeRequest]) (*connect.Response[iamv1.GetMeResponse], error) {
	return connect.NewResponse(&iamv1.GetMeResponse{
		User: &iamv1.UserProto{
			Id:                   req.Msg.UserId,
			TenantId:             req.Msg.TenantId,
			Email:                "admin@mrbiz.local",
			FullName:             "Executive Administrator",
			Role:                 "admin",
			IsSuperAdmin:         true,
			IsActive:             true,
			CurrentBusinessUnitId: "healthcare",
			AllowedBusinessUnits: []string{"healthcare", "travel", "realestate", "ecommerce", "consulting"},
		},
		Tenant: &iamv1.TenantProto{
			Id:                 req.Msg.TenantId,
			Name:               "Global Business Group",
			LegalName:          "Global Business Group Ltd",
			Currency:           "INR",
			SubscriptionPlan:   "enterprise",
			SubscriptionStatus: "active",
			IsActive:           true,
		},
	}), nil
}

func (s *Server) ValidateSession(ctx context.Context, req *connect.Request[iamv1.ValidateSessionRequest]) (*connect.Response[iamv1.ValidateSessionResponse], error) {
	claims, err := tokens.ValidateToken(req.Msg.Token)
	if err != nil {
		return connect.NewResponse(&iamv1.ValidateSessionResponse{IsValid: false}), nil
	}

	return connect.NewResponse(&iamv1.ValidateSessionResponse{
		IsValid: true,
		User: &iamv1.UserProto{
			Id:                   claims.UserID,
			TenantId:             "tenant-1",
			Email:                "admin@mrbiz.local",
			FullName:             "Executive Administrator",
			Role:                 claims.Role,
			IsSuperAdmin:         claims.Role == "admin",
			IsActive:             true,
			CurrentBusinessUnitId: "healthcare",
			AllowedBusinessUnits: []string{"healthcare", "travel", "realestate", "ecommerce", "consulting"},
		},
		Tenant: &iamv1.TenantProto{
			Id:                 "tenant-1",
			Name:               "Global Business Group",
			LegalName:          "Global Business Group Ltd",
			Currency:           "INR",
			SubscriptionPlan:   "enterprise",
			SubscriptionStatus: "active",
			IsActive:           true,
		},
	}), nil
}

func (s *Server) SwitchBusinessUnit(ctx context.Context, req *connect.Request[iamv1.SwitchBusinessUnitRequest]) (*connect.Response[iamv1.SwitchBusinessUnitResponse], error) {
	target := req.Msg.TargetBusinessUnitId
	newToken, err := tokens.CreateAccessToken("user-1", "admin", nil, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&iamv1.SwitchBusinessUnitResponse{
		Success:              true,
		ActiveBusinessUnitId: target,
		UpdatedToken:         newToken,
	}), nil
}

func (s *Server) ListAccessibleUnits(ctx context.Context, req *connect.Request[iamv1.ListAccessibleUnitsRequest]) (*connect.Response[iamv1.ListAccessibleUnitsResponse], error) {
	units := []*iamv1.BusinessUnitProto{
		{Id: "travel", Name: "Travel, Tours & Visa", Code: "TRAVEL", Tagline: "Destinations & Immigration", Color: "#3b82f6", IsActive: true},
		{Id: "healthcare", Name: "Smart Healthcare", Code: "HEALTH", Tagline: "Clinics, Doctors & Telehealth", Color: "#10b981", IsActive: true},
		{Id: "realestate", Name: "Real Estate & Land", Code: "REALESTATE", Tagline: "Property Inventory & Deals", Color: "#f59e0b", IsActive: true},
		{Id: "ecommerce", Name: "E-Commerce & POD", Code: "ECOM", Tagline: "Print On Demand & Logistics", Color: "#8b5cf6", IsActive: true},
		{Id: "consulting", Name: "Software Consulting", Code: "CONSULTING", Tagline: "SOWs, Sprints & Engineering", Color: "#ec4899", IsActive: true},
	}
	return connect.NewResponse(&iamv1.ListAccessibleUnitsResponse{
		Units:        units,
		ActiveUnitId: "healthcare",
	}), nil
}

func (s *Server) GetCompany(ctx context.Context, req *connect.Request[iamv1.GetCompanyRequest]) (*connect.Response[iamv1.GetCompanyResponse], error) {
	return connect.NewResponse(&iamv1.GetCompanyResponse{
		Tenant: &iamv1.TenantProto{
			Id:                 req.Msg.TenantId,
			Name:               "Global Business Group",
			LegalName:          "Global Business Group Ltd",
			Currency:           "INR",
			SubscriptionPlan:   "enterprise",
			SubscriptionStatus: "active",
			IsActive:           true,
		},
	}), nil
}

func (s *Server) UpdateCompany(ctx context.Context, req *connect.Request[iamv1.UpdateCompanyRequest]) (*connect.Response[iamv1.UpdateCompanyResponse], error) {
	return connect.NewResponse(&iamv1.UpdateCompanyResponse{
		Tenant: &iamv1.TenantProto{
			Id:        req.Msg.TenantId,
			Name:      req.Msg.Name,
			LegalName: req.Msg.LegalName,
			IsActive:  true,
		},
	}), nil
}

func (s *Server) ListTenants(ctx context.Context, req *connect.Request[iamv1.ListTenantsRequest]) (*connect.Response[iamv1.ListTenantsResponse], error) {
	return connect.NewResponse(&iamv1.ListTenantsResponse{
		Tenants: []*iamv1.TenantProto{
			{
				Id:                 "tenant-1",
				Name:               "Global Business Group",
				LegalName:          "Global Business Group Ltd",
				Currency:           "INR",
				SubscriptionPlan:   "enterprise",
				SubscriptionStatus: "active",
				IsActive:           true,
			},
		},
	}), nil
}

func (s *Server) CreateTenant(ctx context.Context, req *connect.Request[iamv1.CreateTenantRequest]) (*connect.Response[iamv1.CreateTenantResponse], error) {
	return connect.NewResponse(&iamv1.CreateTenantResponse{
		Tenant: &iamv1.TenantProto{
			Id:               "new-tenant-id",
			Name:             req.Msg.Name,
			LegalName:        req.Msg.LegalName,
			Gstin:            req.Msg.Gstin,
			Email:            req.Msg.Email,
			Phone:            req.Msg.Phone,
			State:            req.Msg.State,
			SubscriptionPlan: req.Msg.SubscriptionPlan,
			IsActive:         true,
		},
	}), nil
}

// Compile-time check
var _ iamv1connect.IAMServiceHandler = (*Server)(nil)
