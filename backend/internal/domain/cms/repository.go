package cms

import (
	"gorm.io/gorm"
)

type Repository interface {
	CreateSite(site *SiteConfig) error
	GetSiteByDomain(domain string) (*SiteConfig, error)
	GetSiteByID(id uint) (*SiteConfig, error)
	ListSites() ([]SiteConfig, error)
	CreatePage(page *CMSPage) error
	GetPageBySlug(siteID uint, slug string) (*CMSPage, error)
	ListPages(siteID uint) ([]CMSPage, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateSite(site *SiteConfig) error {
	return r.db.Create(site).Error
}

func (r *repository) GetSiteByDomain(domain string) (*SiteConfig, error) {
	var site SiteConfig
	err := r.db.Where("domain = ? AND is_active = ?", domain, true).First(&site).Error
	if err != nil {
		return nil, err
	}
	return &site, nil
}

func (r *repository) GetSiteByID(id uint) (*SiteConfig, error) {
	var site SiteConfig
	err := r.db.First(&site, id).Error
	if err != nil {
		return nil, err
	}
	return &site, nil
}

func (r *repository) ListSites() ([]SiteConfig, error) {
	var sites []SiteConfig
	err := r.db.Where("is_active = ?", true).Order("brand_name ASC").Find(&sites).Error
	return sites, err
}

func (r *repository) CreatePage(page *CMSPage) error {
	return r.db.Create(page).Error
}

func (r *repository) GetPageBySlug(siteID uint, slug string) (*CMSPage, error) {
	var page CMSPage
	err := r.db.Where("site_id = ? AND slug = ? AND is_published = ?", siteID, slug, true).First(&page).Error
	if err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *repository) ListPages(siteID uint) ([]CMSPage, error) {
	var pages []CMSPage
	q := r.db.Where("is_published = ?", true)
	if siteID > 0 {
		q = q.Where("site_id = ?", siteID)
	}
	err := q.Order("slug ASC").Find(&pages).Error
	return pages, err
}
