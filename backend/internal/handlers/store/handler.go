package store

import (
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/services/store"
	"drayya-core/internal/utils/response"
	"strconv"

	"drayya-core/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// SearchProducts godoc
// @Summary Search for products
// @Description Search products with filters like category, search term, brand, and price range
// @Tags Store
// @Produce json
// @Param category query string false "Category"
// @Param q query string false "Search query"
// @Param brand query string false "Brand"
// @Param min_price query number false "Min Price"
// @Param max_price query number false "Max Price"
// @Success 200 {array} pharmacy.PharmacyInventory
// @Router /store/products [get]
func SearchProducts(c *fiber.Ctx) error {
	category := c.Query("category")
	search := c.Query("q")
	brand := c.Query("brand")
	sort := c.Query("sort", "price_asc")
	limit := 50
	offset := 0

	minPrice, _ := strconv.ParseFloat(c.Query("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(c.Query("max_price"), 64)

	if l := c.Query("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	if o := c.Query("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}

	products, err := store.SearchProducts(category, search, brand, minPrice, maxPrice, sort, limit, offset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PRODUCTS_FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, products, "Fetched")
}

// GetProduct godoc
// @Summary Get product details
// @Description Fetch details of a single product by ID
// @Tags Store
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} pharmacy.PharmacyInventory
// @Failure 404 {object} map[string]string
// @Router /store/products/{id} [get]
func GetProduct(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	product, err := store.GetProductDetails(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "PRODUCT_NOT_FOUND", "Product not found")
	}
	return response.JSON(c, fiber.StatusOK, product, "Fetched")
}

// ListCategories returns all unique product categories
func ListCategories(c *fiber.Ctx) error {
	cats, err := store.GetCategories()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CATEGORIES_FETCH_FAILED", "Failed to fetch categories")
	}
	return response.JSON(c, fiber.StatusOK, cats, "Fetched")
}

// CreateProduct handler for admins
func CreateProduct(c *fiber.Ctx) error {
	var product pharmacy.PharmacyInventory
	if err := c.BodyParser(&product); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	res, err := store.CreateProduct(product)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PRODUCT_CREATION_FAILED", "Creation failed")
	}

	return response.JSON(c, fiber.StatusCreated, res, "Created")
}

// UpdateProduct handler
func UpdateProduct(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	if err := store.UpdateProduct(uint(id), updates); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PRODUCT_UPDATE_FAILED", "Update failed")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Product updated")
}

// DeleteProduct handler
func DeleteProduct(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := store.DeleteProduct(uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PRODUCT_DELETE_FAILED", "Delete failed")
	}
	return response.JSON(c, fiber.StatusOK, nil, "Product deleted")
}

func RegisterStoreRoutes(router fiber.Router) {
	storeGroup := router.Group("/store")
	storeGroup.Get("/products", SearchProducts)
	storeGroup.Get("/products/:id", GetProduct)
	storeGroup.Get("/categories", ListCategories)

	// Protected Store Routes
	protected := storeGroup.Group("/", middleware.Protected())
	protected.Post("/products", CreateProduct)
	protected.Put("/products/:id", UpdateProduct)
	protected.Delete("/products/:id", DeleteProduct)
	protected.Get("/orders", GetOrders)
	protected.Post("/orders", CreateOrder)
	protected.Get("/delivery-check", CheckDelivery)

	// Retail (Legacy Aliases)
	retailGroup := router.Group("/retail")
	retailGroup.Get("/products", SearchProducts)
	retailGroup.Post("/orders", middleware.Protected(), CreateOrder)
	retailGroup.Get("/inventory", middleware.Protected(), func(c *fiber.Ctx) error { return c.JSON([]fiber.Map{}) })                                    // stub
	retailGroup.Post("/sales", middleware.Protected(), func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"message": "Sale processed"}) }) // stub
}
