package realestate

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"
	realestatev1 "drayya-core/gen/realestate/v1"
	"drayya-core/gen/realestate/v1/realestatev1connect"
	"drayya-core/internal/authctx"
	"drayya-core/internal/database"
	redomain "drayya-core/internal/domain/realestate"
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

const reBusinessUnitID = 3 // Real Estate

func (s *Server) ListProperties(ctx context.Context, req *connect.Request[realestatev1.ListPropertiesRequest]) (*connect.Response[realestatev1.ListPropertiesResponse], error) {
	var props []*realestatev1.PropertyProto

	// Fail closed: no authorized tenant -> no org scope -> return nothing.
	org := orgID(ctx)
	tenantStr := authctx.TenantID(ctx)

	if s.db != nil && org != 0 {
		var dbProps []redomain.Property
		s.db.Scopes(database.OrgScope(org)).Find(&dbProps)
		if len(dbProps) == 0 {
			// Seed default properties for THIS tenant only.
			p1 := redomain.Property{
				OrganizationID:  org,
				BusinessUnitID:  reBusinessUnitID,
				Title:           "Emerald Heights Luxury 3BHK",
				PropertyType:    "APARTMENT",
				TransactionType: "SALE",
				Price:           18500000.00,
				City:            "Hyderabad",
				Location:        "Banjara Hills, Hyderabad",
				Bedrooms:        3,
				AreaSqFt:        2450,
				Status:          "AVAILABLE",
				Amenities:       `["Swimming Pool","Clubhouse","EV Charging","24/7 Security"]`,
			}
			p2 := redomain.Property{
				OrganizationID:  org,
				BusinessUnitID:  reBusinessUnitID,
				Title:           "Skyline Commercial Hub Unit 402",
				PropertyType:    "COMMERCIAL",
				TransactionType: "RENT",
				Price:           32000000.00,
				City:            "Hyderabad",
				Location:        "Hitec City, Hyderabad",
				Bedrooms:        0,
				AreaSqFt:        3800,
				Status:          "AVAILABLE",
				Amenities:       `["Grade A Office","100% Power Backup","High Speed Elevators"]`,
			}
			s.db.Create(&p1)
			s.db.Create(&p2)
			dbProps = []redomain.Property{p1, p2}
		}

		for _, p := range dbProps {
			props = append(props, &realestatev1.PropertyProto{
				Id:            fmt.Sprintf("prop-%d", p.ID),
				TenantId:      tenantStr,
				Title:         p.Title,
				PropertyType:  p.PropertyType,
				Location:      p.Location,
				Price:         fmt.Sprintf("%.2f", p.Price),
				AreaSqft:      fmt.Sprintf("%.0f", p.AreaSqFt),
				Bedrooms:      int32(p.Bedrooms),
				Status:        p.Status,
				AmenitiesJson: p.Amenities,
			})
		}
	}

	return connect.NewResponse(&realestatev1.ListPropertiesResponse{Properties: props}), nil
}

func (s *Server) GetProperty(ctx context.Context, req *connect.Request[realestatev1.GetPropertyRequest]) (*connect.Response[realestatev1.GetPropertyResponse], error) {
	org := orgID(ctx)
	if s.db == nil || org == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("tenant context required"))
	}
	var p redomain.Property
	// Scoped lookup: a property id from another tenant resolves to not-found.
	if err := s.db.Scopes(database.OrgScope(org)).First(&p, "id = ?", req.Msg.Id).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("property not found: %s", req.Msg.Id))
	}
	return connect.NewResponse(&realestatev1.GetPropertyResponse{
		Property: &realestatev1.PropertyProto{
			Id:            fmt.Sprintf("prop-%d", p.ID),
			TenantId:      authctx.TenantID(ctx),
			Title:         p.Title,
			PropertyType:  p.PropertyType,
			Location:      p.Location,
			Price:         fmt.Sprintf("%.2f", p.Price),
			AreaSqft:      fmt.Sprintf("%.0f", p.AreaSqFt),
			Bedrooms:      int32(p.Bedrooms),
			Status:        p.Status,
			AmenitiesJson: p.Amenities,
		},
	}), nil
}

func (s *Server) CreateProperty(ctx context.Context, req *connect.Request[realestatev1.CreatePropertyRequest]) (*connect.Response[realestatev1.CreatePropertyResponse], error) {
	org := orgID(ctx)
	if org == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("tenant context required"))
	}
	priceVal, _ := strconv.ParseFloat(req.Msg.Price, 64)
	areaVal, _ := strconv.ParseFloat(req.Msg.AreaSqft, 64)

	newProp := redomain.Property{
		OrganizationID:  org,
		BusinessUnitID:  reBusinessUnitID,
		Title:           req.Msg.Title,
		PropertyType:    req.Msg.PropertyType,
		TransactionType: "SALE",
		Price:           priceVal,
		City:            "Hyderabad",
		Location:        req.Msg.Location,
		Bedrooms:        int(req.Msg.Bedrooms),
		AreaSqFt:        areaVal,
		Status:          "AVAILABLE",
	}

	if s.db != nil {
		_ = s.db.Create(&newProp).Error
	}

	idStr := fmt.Sprintf("prop-%d", newProp.ID)
	if newProp.ID == 0 {
		idStr = fmt.Sprintf("prop-%d", time.Now().Unix())
	}

	prop := &realestatev1.PropertyProto{
		Id:           idStr,
		TenantId:     authctx.TenantID(ctx),
		Title:        req.Msg.Title,
		PropertyType: req.Msg.PropertyType,
		Location:     req.Msg.Location,
		Price:        req.Msg.Price,
		AreaSqft:     req.Msg.AreaSqft,
		Bedrooms:     req.Msg.Bedrooms,
		Status:       "AVAILABLE",
	}
	return connect.NewResponse(&realestatev1.CreatePropertyResponse{Property: prop}), nil
}

func (s *Server) ScheduleSiteVisit(ctx context.Context, req *connect.Request[realestatev1.ScheduleSiteVisitRequest]) (*connect.Response[realestatev1.ScheduleSiteVisitResponse], error) {
	propID, _ := strconv.ParseUint(req.Msg.PropertyId, 10, 64)
	visitTime, err := time.Parse(time.RFC3339, req.Msg.ScheduledTime)
	if err != nil {
		visitTime = time.Now().Add(24 * time.Hour)
	}

	org := orgID(ctx)
	if org == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("tenant context required"))
	}
	newVisit := redomain.SiteVisit{
		OrganizationID: org,
		BusinessUnitID: reBusinessUnitID,
		PropertyID:     uint(propID),
		BuyerID:        1,
		AgentID:        1,
		ScheduledAt:    visitTime,
		Status:         "SCHEDULED",
		Notes:          fmt.Sprintf("%s (%s)", req.Msg.ClientName, req.Msg.ClientPhone),
	}

	if s.db != nil {
		_ = s.db.Create(&newVisit).Error
	}

	idStr := fmt.Sprintf("visit-%d", newVisit.ID)
	if newVisit.ID == 0 {
		idStr = fmt.Sprintf("visit-%d", time.Now().Unix())
	}

	visit := &realestatev1.SiteVisitProto{
		Id:            idStr,
		TenantId:      authctx.TenantID(ctx),
		PropertyId:    req.Msg.PropertyId,
		ClientName:    req.Msg.ClientName,
		ClientPhone:   req.Msg.ClientPhone,
		ScheduledTime: req.Msg.ScheduledTime,
		Status:        "SCHEDULED",
	}
	return connect.NewResponse(&realestatev1.ScheduleSiteVisitResponse{SiteVisit: visit}), nil
}

func (s *Server) ListSiteVisits(ctx context.Context, req *connect.Request[realestatev1.ListSiteVisitsRequest]) (*connect.Response[realestatev1.ListSiteVisitsResponse], error) {
	var visits []*realestatev1.SiteVisitProto

	org := orgID(ctx)
	tenantStr := authctx.TenantID(ctx)

	if s.db != nil && org != 0 {
		var dbVisits []redomain.SiteVisit
		s.db.Scopes(database.OrgScope(org)).Find(&dbVisits)
		if len(dbVisits) == 0 {
			v1 := redomain.SiteVisit{
				OrganizationID: org,
				BusinessUnitID: reBusinessUnitID,
				PropertyID:     1,
				BuyerID:        1,
				AgentID:        1,
				ScheduledAt:    time.Now().Add(24 * time.Hour),
				Status:         "SCHEDULED",
				Notes:          "Vikram Reddy (+91 98490 12345)",
			}
			s.db.Create(&v1)
			dbVisits = []redomain.SiteVisit{v1}
		}

		for _, v := range dbVisits {
			visits = append(visits, &realestatev1.SiteVisitProto{
				Id:            fmt.Sprintf("visit-%d", v.ID),
				TenantId:      tenantStr,
				PropertyId:    fmt.Sprintf("%d", v.PropertyID),
				ClientName:    v.Notes,
				ClientPhone:   "+91 98490 12345",
				ScheduledTime: v.ScheduledAt.Format(time.RFC3339),
				Status:        v.Status,
			})
		}
	}

	return connect.NewResponse(&realestatev1.ListSiteVisitsResponse{SiteVisits: visits}), nil
}

func (s *Server) CreatePropertyDeal(ctx context.Context, req *connect.Request[realestatev1.CreatePropertyDealRequest]) (*connect.Response[realestatev1.CreatePropertyDealResponse], error) {
	propID, _ := strconv.ParseUint(req.Msg.PropertyId, 10, 64)
	dealAmt, _ := strconv.ParseFloat(req.Msg.DealAmount, 64)
	commAmt, _ := strconv.ParseFloat(req.Msg.CommissionAmount, 64)

	org := orgID(ctx)
	if org == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("tenant context required"))
	}
	newDeal := redomain.PropertyDeal{
		OrganizationID:   org,
		BusinessUnitID:   reBusinessUnitID,
		PropertyID:       uint(propID),
		BuyerID:          1,
		SellerID:         2,
		AgreedPrice:      dealAmt,
		CommissionAmount: commAmt,
		Status:           "TOKEN_PAID",
	}

	if s.db != nil {
		_ = s.db.Create(&newDeal).Error
	}

	idStr := fmt.Sprintf("deal-%d", newDeal.ID)
	if newDeal.ID == 0 {
		idStr = fmt.Sprintf("deal-%d", time.Now().Unix())
	}

	deal := &realestatev1.PropertyDealProto{
		Id:               idStr,
		TenantId:         authctx.TenantID(ctx),
		PropertyId:       req.Msg.PropertyId,
		BuyerName:        req.Msg.BuyerName,
		SellerName:       req.Msg.SellerName,
		DealAmount:       req.Msg.DealAmount,
		CommissionAmount: req.Msg.CommissionAmount,
		Status:           "TOKEN_PAID",
	}
	return connect.NewResponse(&realestatev1.CreatePropertyDealResponse{Deal: deal}), nil
}

func (s *Server) ListPropertyDeals(ctx context.Context, req *connect.Request[realestatev1.ListPropertyDealsRequest]) (*connect.Response[realestatev1.ListPropertyDealsResponse], error) {
	var deals []*realestatev1.PropertyDealProto

	org := orgID(ctx)
	tenantStr := authctx.TenantID(ctx)

	if s.db != nil && org != 0 {
		var dbDeals []redomain.PropertyDeal
		s.db.Scopes(database.OrgScope(org)).Find(&dbDeals)
		if len(dbDeals) == 0 {
			d1 := redomain.PropertyDeal{
				OrganizationID:   org,
				BusinessUnitID:   reBusinessUnitID,
				PropertyID:       1,
				BuyerID:          1,
				SellerID:         2,
				AgreedPrice:      18000000.00,
				CommissionAmount: 360000.00,
				Status:           "TOKEN_PAID",
			}
			s.db.Create(&d1)
			dbDeals = []redomain.PropertyDeal{d1}
		}

		for _, d := range dbDeals {
			deals = append(deals, &realestatev1.PropertyDealProto{
				Id:               fmt.Sprintf("deal-%d", d.ID),
				TenantId:         tenantStr,
				PropertyId:       fmt.Sprintf("%d", d.PropertyID),
				BuyerName:        "Suresh Nair",
				SellerName:       "Dev Developers",
				DealAmount:       fmt.Sprintf("%.2f", d.AgreedPrice),
				CommissionAmount: fmt.Sprintf("%.2f", d.CommissionAmount),
				Status:           d.Status,
			})
		}
	}

	return connect.NewResponse(&realestatev1.ListPropertyDealsResponse{Deals: deals}), nil
}

// Compile-time check
var _ realestatev1connect.RealEstateServiceHandler = (*Server)(nil)
