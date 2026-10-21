package blog

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/content"
)

func CreateBlog(blog *content.Blog) error {
	return database.DB.Create(blog).Error
}

func GetPublicBlogs(category, search string) ([]content.Blog, error) {
	var blogs []content.Blog
	query := database.DB.Preload("Author").Where("status = ?", "approved")

	if category != "" && category != "All" {
		query = query.Where("category = ?", category)
	}

	if search != "" {
		query = query.Where("title ILIKE ? OR excerpt ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	err := query.Order("created_at desc").Find(&blogs).Error
	return blogs, err
}

func GetPendingBlogs() ([]content.Blog, error) {
	var blogs []content.Blog
	err := database.DB.Preload("Author").Where("status = ?", "pending").Order("created_at asc").Find(&blogs).Error
	return blogs, err
}

func GetBlogByID(id uint) (*content.Blog, error) {
	var blog content.Blog
	err := database.DB.Preload("Author").First(&blog, id).Error
	return &blog, err
}

func UpdateBlogStatus(blogID uint, approverID uint, status string) error {
	return database.DB.Model(&content.Blog{}).Where("id = ?", blogID).Updates(map[string]interface{}{
		"status":      status,
		"approver_id": approverID,
	}).Error
}
