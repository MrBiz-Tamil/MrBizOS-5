package cms

import (
	"errors"
)

type Service interface {
	CreateSite(site *SiteConfig) (*SiteConfig, error)
	GetSiteByDomain(domain string) (*SiteConfig, error)
	GetSiteByID(id uint) (*SiteConfig, error)
	ListSites() ([]SiteConfig, error)
	CreatePage(page *CMSPage) (*CMSPage, error)
	GetPageBySlug(siteID uint, slug string) (*CMSPage, error)
	ListPages(siteID uint) ([]CMSPage, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateSite(site *SiteConfig) (*SiteConfig, error) {
	if site.Domain == "" || site.BrandName == "" {
		return nil, errors.New("domain and brand_name are required")
	}
	if err := s.repo.CreateSite(site); err != nil {
		return nil, err
	}
	return site, nil
}

func (s *service) GetSiteByDomain(domain string) (*SiteConfig, error) {
	return s.repo.GetSiteByDomain(domain)
}

func (s *service) GetSiteByID(id uint) (*SiteConfig, error) {
	return s.repo.GetSiteByID(id)
}

func (s *service) ListSites() ([]SiteConfig, error) {
	return s.repo.ListSites()
}

func (s *service) CreatePage(page *CMSPage) (*CMSPage, error) {
	if page.SiteID == 0 || page.Slug == "" || page.Title == "" {
		return nil, errors.New("site_id, slug, and title are required")
	}
	if err := s.repo.CreatePage(page); err != nil {
		return nil, err
	}
	return page, nil
}

func (s *service) GetPageBySlug(siteID uint, slug string) (*CMSPage, error) {
	return s.repo.GetPageBySlug(siteID, slug)
}

func (s *service) ListPages(siteID uint) ([]CMSPage, error) {
	return s.repo.ListPages(siteID)
}
