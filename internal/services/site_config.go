package services

import (
	"errors"

	"bakaray/internal/config"
	"bakaray/internal/models"
	"gorm.io/gorm"
)

type SiteConfigService struct {
	db *gorm.DB
}

func NewSiteConfigService(db *gorm.DB) *SiteConfigService {
	return &SiteConfigService{db: db}
}

func (s *SiteConfigService) GetOrCreate() (*models.SiteConfig, error) {
	var existing models.SiteConfig
	if err := s.db.Order("id asc").First(&existing).Error; err == nil {
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	site := models.SiteConfig{
		SiteName:           cfg.Site.Name,
		SiteDomain:         cfg.Site.Domain,
		NodeSecret:         cfg.Site.NodeSecret,
		NodeReportInterval: cfg.Site.NodeReportInterval,
	}
	if site.SiteName == "" {
		site.SiteName = "BakaRay"
	}
	if site.NodeReportInterval <= 0 {
		site.NodeReportInterval = 30
	}

	if err := s.db.Create(&site).Error; err != nil {
		return nil, err
	}
	return &site, nil
}

func (s *SiteConfigService) Update(updates map[string]any) (*models.SiteConfig, error) {
	site, err := s.GetOrCreate()
	if err != nil {
		return nil, err
	}

	if err := s.db.Model(site).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := s.db.First(site, site.ID).Error; err != nil {
		return nil, err
	}
	return site, nil
}
