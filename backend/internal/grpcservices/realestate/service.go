package realestate

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	realestatev1 "drayya-core/gen/realestate/v1"
	"drayya-core/gen/realestate/v1/realestatev1connect"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ListProperties(ctx context.Context, req *connect.Request[realestatev1.ListPropertiesRequest]) (*connect.Response[realestatev1.ListPropertiesResponse], error) {
	props := []*realestatev1.PropertyProto{
		{
			Id:             "prop-1",
			TenantId:       "tenant-1",
			Title:          "Emerald Heights Luxury 3BHK",
			PropertyType:   "APARTMENT",
			Location:       "Banjara Hills, Hyderabad",
			Price:          "18500000.00",
			AreaSqft:       "2450",
			Bedrooms:       3,
			Status:         "AVAILABLE",
			AmenitiesJson:  `["Swimming Pool","Clubhouse","EV Charging","24/7 Security"]`,
		},
		{
			Id:             "prop-2",
			TenantId:       "tenant-1",
			Title:          "Skyline Commercial Hub Unit 402",
			PropertyType:   "COMMERCIAL",
			Location:       "Hitec City, Hyderabad",
			Price:          "32000000.00",
			AreaSqft:       "3800",
			Bedrooms:       0,
			Status:         "AVAILABLE",
			AmenitiesJson:  `["Grade A Office","100% Power Backup","High Speed Elevators"]`,
		},
	}
	return connect.NewResponse(&realestatev1.ListPropertiesResponse{Properties: props}), nil
}

func (s *Server) GetProperty(ctx context.Context, req *connect.Request[realestatev1.GetPropertyRequest]) (*connect.Response[realestatev1.GetPropertyResponse], error) {
	return connect.NewResponse(&realestatev1.GetPropertyResponse{
		Property: &realestatev1.PropertyProto{
			Id:           req.Msg.Id,
			TenantId:     "tenant-1",
			Title:        "Emerald Heights Luxury 3BHK",
			PropertyType: "APARTMENT",
			Location:     "Banjara Hills, Hyderabad",
			Price:        "18500000.00",
			AreaSqft:     "2450",
			Bedrooms:     3,
			Status:       "AVAILABLE",
		},
	}), nil
}

func (s *Server) CreateProperty(ctx context.Context, req *connect.Request[realestatev1.CreatePropertyRequest]) (*connect.Response[realestatev1.CreatePropertyResponse], error) {
	prop := &realestatev1.PropertyProto{
		Id:           fmt.Sprintf("prop-%d", time.Now().Unix()),
		TenantId:     "tenant-1",
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
	visit := &realestatev1.SiteVisitProto{
		Id:            fmt.Sprintf("visit-%d", time.Now().Unix()),
		TenantId:      "tenant-1",
		PropertyId:    req.Msg.PropertyId,
		ClientName:    req.Msg.ClientName,
		ClientPhone:   req.Msg.ClientPhone,
		ScheduledTime: req.Msg.ScheduledTime,
		Status:        "SCHEDULED",
	}
	return connect.NewResponse(&realestatev1.ScheduleSiteVisitResponse{SiteVisit: visit}), nil
}

func (s *Server) ListSiteVisits(ctx context.Context, req *connect.Request[realestatev1.ListSiteVisitsRequest]) (*connect.Response[realestatev1.ListSiteVisitsResponse], error) {
	visits := []*realestatev1.SiteVisitProto{
		{
			Id:            "visit-1",
			TenantId:      "tenant-1",
			PropertyId:    req.Msg.PropertyId,
			ClientName:    "Vikram Reddy",
			ClientPhone:   "+91 98490 12345",
			ScheduledTime: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			Status:        "SCHEDULED",
		},
	}
	return connect.NewResponse(&realestatev1.ListSiteVisitsResponse{SiteVisits: visits}), nil
}

func (s *Server) CreatePropertyDeal(ctx context.Context, req *connect.Request[realestatev1.CreatePropertyDealRequest]) (*connect.Response[realestatev1.CreatePropertyDealResponse], error) {
	deal := &realestatev1.PropertyDealProto{
		Id:               fmt.Sprintf("deal-%d", time.Now().Unix()),
		TenantId:         "tenant-1",
		PropertyId:       req.Msg.PropertyId,
		BuyerName:        req.Msg.BuyerName,
		SellerName:       req.Msg.SellerName,
		DealAmount:       req.Msg.DealAmount,
		CommissionAmount: req.Msg.CommissionAmount,
		Status:           "NEGOTIATION",
	}
	return connect.NewResponse(&realestatev1.CreatePropertyDealResponse{Deal: deal}), nil
}

func (s *Server) ListPropertyDeals(ctx context.Context, req *connect.Request[realestatev1.ListPropertyDealsRequest]) (*connect.Response[realestatev1.ListPropertyDealsResponse], error) {
	deals := []*realestatev1.PropertyDealProto{
		{
			Id:               "deal-1",
			TenantId:         "tenant-1",
			PropertyId:       "prop-1",
			BuyerName:        "Suresh Nair",
			SellerName:       "Dev Developers",
			DealAmount:       "18000000.00",
			CommissionAmount: "360000.00",
			Status:           "TOKEN_PAID",
		},
	}
	return connect.NewResponse(&realestatev1.ListPropertyDealsResponse{Deals: deals}), nil
}

// Compile-time check
var _ realestatev1connect.RealEstateServiceHandler = (*Server)(nil)
