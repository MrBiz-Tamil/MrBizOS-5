package integration

import (
	"testing"

	"drayya-core/internal/domain/cms"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCMSTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to memory db: %v", err)
	}

	if err := cms.AutoMigrate(db); err != nil {
		t.Fatalf("cms automigrate failed: %v", err)
	}

	return db
}

func TestMultiBrandCMSSiteResolver(t *testing.T) {
	db := setupCMSTestDB(t)
	svc := cms.NewService(cms.NewRepository(db))

	// 1. Create Site Configuration for Travel Brand
	travelSite, err := svc.CreateSite(&cms.SiteConfig{
		OrganizationID: 1,
		BusinessUnitID: 1, // Travel
		Domain:         "travel-expeditions.com",
		BrandName:      "Apex Expeditions & Visa Services",
		PrimaryColor:   "#0284c7",
		SecondaryColor: "#0f172a",
		FontFamily:     "Outfit",
	})
	if err != nil || travelSite.ID == 0 {
		t.Fatalf("CreateSite failed: %v", err)
	}

	// 2. Resolve Site by Domain
	resolved, err := svc.GetSiteByDomain("travel-expeditions.com")
	if err != nil || resolved.BrandName != "Apex Expeditions & Visa Services" {
		t.Fatalf("GetSiteByDomain failed: %v", err)
	}
	if resolved.PrimaryColor != "#0284c7" {
		t.Fatalf("expected primary color #0284c7, got %s", resolved.PrimaryColor)
	}

	// 3. Create CMS Page for Travel Site
	page, err := svc.CreatePage(&cms.CMSPage{
		SiteID:       travelSite.ID,
		Slug:         "/tours/bali",
		Title:        "Bali Luxury Expeditions & Visa Assistance",
		ContentJSON:  `{"sections":[{"type":"hero","title":"Discover Bali"},{"type":"packages"}]}`,
		SEOKeywords:  "bali tours, indonesia visa, luxury travel",
		CanonicalURL: "https://travel-expeditions.com/tours/bali",
	})
	if err != nil || page.ID == 0 {
		t.Fatalf("CreatePage failed: %v", err)
	}

	// 4. Resolve CMS Page by Slug
	resolvedPage, err := svc.GetPageBySlug(travelSite.ID, "/tours/bali")
	if err != nil || resolvedPage.Title != "Bali Luxury Expeditions & Visa Assistance" {
		t.Fatalf("GetPageBySlug failed: %v", err)
	}
}
