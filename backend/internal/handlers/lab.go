//go:build ignore

package handlers

import (
	"drayya-core/internal/lab"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func CreateLabRequest(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Params("hospitalId"))

	var req struct {
		PatientID uint   `json:"patient_id"`
		DoctorID  uint   `json:"doctor_id"`
		TestIDs   []uint `json:"test_ids"`
		Priority  string `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	result, err := lab.CreateLabRequest(uint(hospitalID), req.PatientID, req.DoctorID, req.TestIDs, req.Priority)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LAB_REQUEST_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, result, "Fetched")
}

func GetLabRequests(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Params("hospitalId"))
	status := c.Query("status")

	requests, err := lab.GetLabRequests(uint(hospitalID), status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LAB_REQUESTS_FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, requests, "Fetched")
}

func UpdateSpecimen(c *fiber.Ctx) error {
	requestID, _ := strconv.Atoi(c.Params("requestId"))

	var req struct {
		Type      string `json:"type"`
		Barcode   string `json:"barcode"`
		Status    string `json:"status"`
		Collector uint   `json:"collector_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	err := lab.UpdateSpecimenStatus(uint(requestID), req.Type, req.Barcode, req.Status, req.Collector)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SPECIMEN_UPDATE_FAILED", err.Error())
	}

	return c.SendStatus(200)
}

func SubmitLabResults(c *fiber.Ctx) error {
	requestID, _ := strconv.Atoi(c.Params("requestId"))

	var req struct {
		TechnicianID uint                     `json:"technician_id"`
		Results      []map[string]interface{} `json:"results"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	err := lab.SubmitLabResults(uint(requestID), req.TechnicianID, req.Results)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SUBMIT_RESULTS_FAILED", err.Error())
	}

	return c.SendStatus(200)
}

func GetLabTests(c *fiber.Ctx) error {
	tests, err := lab.GetLabTests()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LAB_TESTS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, tests, "Fetched")
}

func RegisterLabRoutes(router fiber.Router) {
	labGroup := router.Group("/lab")

	labGroup.Get("/tests", GetLabTests)
	labGroup.Post("/request/:hospitalId", CreateLabRequest)
	labGroup.Get("/requests/:hospitalId", GetLabRequests)
	labGroup.Post("/specimen/:requestId", UpdateSpecimen)
	labGroup.Post("/results/:requestId", SubmitLabResults)
}
