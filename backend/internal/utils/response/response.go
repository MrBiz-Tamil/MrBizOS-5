package response

import (
	"github.com/gofiber/fiber/v2"
)

// StandardResponse follows the defined success envelope
type StandardResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// PaginationMeta contains standardized pagination metadata
type PaginationMeta struct {
	Total       int64 `json:"total"`
	Page        int   `json:"page"`
	Limit       int   `json:"limit"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

// StandardError follows the defined error envelope
type StandardError struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// JSON sends a standardized success response
func JSON(c *fiber.Ctx, status int, data interface{}, message string, meta ...interface{}) error {
	resp := StandardResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	if len(meta) > 0 {
		resp.Meta = meta[0]
	}
	return c.Status(status).JSON(resp)
}

// Error sends a standardized error response
func Error(c *fiber.Ctx, status int, code string, message string, details ...interface{}) error {
	errResp := StandardError{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
	if len(details) > 0 {
		errResp.Error.Details = details[0]
	}
	return c.Status(status).JSON(errResp)
}

// NewPaginationMeta creates a standardized pagination meta object
func NewPaginationMeta(total int64, page, limit int) PaginationMeta {
	totalPages := int(total / int64(limit))
	if total%int64(limit) > 0 {
		totalPages++
	}

	return PaginationMeta{
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}
