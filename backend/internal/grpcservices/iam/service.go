package iam

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	iamv1 "drayya-core/gen/iam/v1"
	"drayya-core/gen/iam/v1/iamv1connect"
	"drayya-core/internal/authctx"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/tokens"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Server struct {
	db   *gorm.DB
	repo identity.Repository
}

func NewServer(db *gorm.DB) *Server {
	var repo identity.Repository
	if db != nil {
		repo = identity.NewRepository(db)
	}
	return &Server{db: db, repo: repo}
}

// userProto builds the IAM UserProto from a persisted identity.User, deriving the
// tenant string and BU scope via the documented mappings (mapping.go). It never
// fabricates privilege: IsSuperAdmin is true only for the platform "superadmin"
// role, and AllowedBusinessUnits is the user's real scope, not a blanket list.
func userProto(u *identity.User) *iamv1.UserProto {
	return &iamv1.UserProto{
		Id:                    strconv.FormatUint(uint64(u.ID), 10),
		TenantId:              tenantStringForOrg(u.OrganizationID),
		Email:                 u.Email,
		FullName:              u.FullName,
		Role:                  u.Role,
		IsSuperAdmin:          isPlatformSuperAdmin(u.Role),
		IsActive:              u.IsActive,
		CurrentBusinessUnitId: buSlug(u.BusinessUnitID),
		AllowedBusinessUnits:  allowedBUsForRole(u.Role, u.BusinessUnitID),
	}
}

// tenantProto renders the minimal, non-fabricated tenant view for an organization.
// Rich tenant master data (legal name, GSTIN, plan, address) is owned by the
// organization model (tracked as T4) and is intentionally left blank here rather
// than hardcoded to a fictitious "Global Business Group".
func tenantProto(orgID uint) *iamv1.TenantProto {
	if orgID == 0 {
		orgID = 1
	}
	return &iamv1.TenantProto{
		Id:       tenantStringForOrg(orgID),
		Name:     fmt.Sprintf("Organization %d", orgID),
		Currency: "INR",
		IsActive: true,
	}
}

// loadUser resolves a persisted user by its string id.
func (s *Server) loadUser(idStr string) (*identity.User, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("identity store unavailable")
	}
	id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64)
	if err != nil || id == 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	return s.repo.GetUserByID(uint(id))
}

func (s *Server) Login(ctx context.Context, req *connect.Request[iamv1.LoginRequest]) (*connect.Response[iamv1.LoginResponse], error) {
	email := strings.TrimSpace(req.Msg.Email)
	password := req.Msg.Password
	if email == "" || password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email and password are required"))
	}
	if s.repo == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("identity store unavailable"))
	}

	// Verify against a real, persisted user. An unknown email, an empty stored
	// hash, or a wrong password all yield the same opaque "invalid credentials"
	// error. There is no admin-email fallback and no empty-hash skip — the prior
	// behaviour that let those become a super-admin session is removed (T1).
	user, err := s.repo.GetUserByEmail(email)
	if err != nil || user == nil || user.PasswordHash == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	if !user.IsActive {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account is inactive"))
	}

	tenant := tenantStringForOrg(user.OrganizationID)
	currentBU := buSlug(user.BusinessUnitID)
	allowed := allowedBUsForRole(user.Role, user.BusinessUnitID)

	token, err := tokens.CreateTenantAccessToken(strconv.FormatUint(uint64(user.ID), 10), tenant, user.Role, currentBU, allowed)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&iamv1.LoginResponse{
		User:   userProto(user),
		Tenant: tenantProto(user.OrganizationID),
		Token:  token,
	}), nil
}

func (s *Server) Logout(ctx context.Context, req *connect.Request[iamv1.LogoutRequest]) (*connect.Response[iamv1.LogoutResponse], error) {
	return connect.NewResponse(&iamv1.LogoutResponse{Message: "Logged out successfully"}), nil
}

func (s *Server) GetMe(ctx context.Context, req *connect.Request[iamv1.GetMeRequest]) (*connect.Response[iamv1.GetMeResponse], error) {
	// Identity comes from the authenticated session (set by the auth interceptor),
	// never from client-supplied request fields.
	user, err := s.loadUser(authctx.UserID(ctx))
	if err != nil || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	return connect.NewResponse(&iamv1.GetMeResponse{
		User:   userProto(user),
		Tenant: tenantProto(user.OrganizationID),
	}), nil
}

func (s *Server) ValidateSession(ctx context.Context, req *connect.Request[iamv1.ValidateSessionRequest]) (*connect.Response[iamv1.ValidateSessionResponse], error) {
	claims, err := tokens.ValidateToken(req.Msg.Token)
	if err != nil {
		return connect.NewResponse(&iamv1.ValidateSessionResponse{IsValid: false}), nil
	}
	// A structurally valid token must still map to a live user.
	user, err := s.loadUser(claims.UserID)
	if err != nil || user == nil || !user.IsActive {
		return connect.NewResponse(&iamv1.ValidateSessionResponse{IsValid: false}), nil
	}
	return connect.NewResponse(&iamv1.ValidateSessionResponse{
		IsValid: true,
		User:    userProto(user),
		Tenant:  tenantProto(user.OrganizationID),
	}), nil
}

func (s *Server) SwitchBusinessUnit(ctx context.Context, req *connect.Request[iamv1.SwitchBusinessUnitRequest]) (*connect.Response[iamv1.SwitchBusinessUnitResponse], error) {
	user, err := s.loadUser(authctx.UserID(ctx))
	if err != nil || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	target := strings.TrimSpace(req.Msg.TargetBusinessUnitId)
	allowed := allowedBUsForRole(user.Role, user.BusinessUnitID)
	ok := false
	for _, bu := range allowed {
		if bu == target {
			ok = true
			break
		}
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("user does not have access to business unit %s", target))
	}

	newToken, err := tokens.CreateTenantAccessToken(
		strconv.FormatUint(uint64(user.ID), 10),
		tenantStringForOrg(user.OrganizationID),
		user.Role,
		target,
		allowed,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&iamv1.SwitchBusinessUnitResponse{
		Success:              true,
		ActiveBusinessUnitId: target,
		UpdatedToken:         newToken,
	}), nil
}

// buCatalog is the static presentation metadata for the business units. Access is
// filtered per user by allowedBUsForRole — the catalog itself is not sensitive.
var buCatalog = []*iamv1.BusinessUnitProto{
	{Id: "travel", Name: "Travel, Tours & Visa", Code: "TRAVEL", Tagline: "Destinations & Immigration", Color: "#3b82f6", IsActive: true},
	{Id: "healthcare", Name: "Smart Healthcare", Code: "HEALTH", Tagline: "Clinics, Doctors & Telehealth", Color: "#10b981", IsActive: true},
	{Id: "realestate", Name: "Real Estate & Land", Code: "REALESTATE", Tagline: "Property Inventory & Deals", Color: "#f59e0b", IsActive: true},
	{Id: "ecommerce", Name: "E-Commerce & POD", Code: "ECOM", Tagline: "Print On Demand & Logistics", Color: "#8b5cf6", IsActive: true},
	{Id: "consulting", Name: "Software Consulting", Code: "CONSULTING", Tagline: "SOWs, Sprints & Engineering", Color: "#ec4899", IsActive: true},
}

func (s *Server) ListAccessibleUnits(ctx context.Context, req *connect.Request[iamv1.ListAccessibleUnitsRequest]) (*connect.Response[iamv1.ListAccessibleUnitsResponse], error) {
	user, err := s.loadUser(authctx.UserID(ctx))
	if err != nil || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	allowed := map[string]bool{}
	for _, bu := range allowedBUsForRole(user.Role, user.BusinessUnitID) {
		allowed[bu] = true
	}
	var units []*iamv1.BusinessUnitProto
	for _, u := range buCatalog {
		if allowed[u.Id] {
			units = append(units, u)
		}
	}
	return connect.NewResponse(&iamv1.ListAccessibleUnitsResponse{
		Units:        units,
		ActiveUnitId: buSlug(user.BusinessUnitID),
	}), nil
}

func (s *Server) GetCompany(ctx context.Context, req *connect.Request[iamv1.GetCompanyRequest]) (*connect.Response[iamv1.GetCompanyResponse], error) {
	// Return the caller's OWN tenant (from the authenticated session), never a
	// client-supplied tenant id. Rich company master data is T4 (org model).
	return connect.NewResponse(&iamv1.GetCompanyResponse{
		Tenant: tenantProto(orgIDFromTenant(authctx.TenantID(ctx))),
	}), nil
}

// UpdateCompany / CreateTenant mutate the organization master, which is not yet
// persisted (tracked as T4). They fail honestly rather than fabricate a success,
// per the engineering rule against reporting simulated results as real.
func (s *Server) UpdateCompany(ctx context.Context, req *connect.Request[iamv1.UpdateCompanyRequest]) (*connect.Response[iamv1.UpdateCompanyResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("organization master updates are not yet implemented (pending T4 org model)"))
}

func (s *Server) ListTenants(ctx context.Context, req *connect.Request[iamv1.ListTenantsRequest]) (*connect.Response[iamv1.ListTenantsResponse], error) {
	// A non-super-admin only ever sees their own tenant. Platform-wide tenant
	// listing is a super-admin operation gated by the authz interceptor and will
	// read the persisted org table under T4; today it returns the caller's tenant.
	return connect.NewResponse(&iamv1.ListTenantsResponse{
		Tenants: []*iamv1.TenantProto{tenantProto(orgIDFromTenant(authctx.TenantID(ctx)))},
	}), nil
}

func (s *Server) CreateTenant(ctx context.Context, req *connect.Request[iamv1.CreateTenantRequest]) (*connect.Response[iamv1.CreateTenantResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("tenant provisioning is not yet implemented (pending T4 org model)"))
}

// Compile-time check
var _ iamv1connect.IAMServiceHandler = (*Server)(nil)
