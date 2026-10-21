package hr

import (
	"errors"
	"time"
)

// SeedDefaultLeaveTypes creates the standard leave types for the organisation.
func SeedDefaultLeaveTypes() error {
	defaults := []LeaveType{
		{Name: "Annual Leave", Code: "AL", MaxDaysPerYear: 18, IsPaid: true, CarryForward: true, MaxCarryDays: 6},
		{Name: "Sick Leave", Code: "SL", MaxDaysPerYear: 12, IsPaid: true, CarryForward: false},
		{Name: "Casual Leave", Code: "CL", MaxDaysPerYear: 6, IsPaid: true, CarryForward: false},
		{Name: "Maternity Leave", Code: "ML", MaxDaysPerYear: 182, IsPaid: true, CarryForward: false},
		{Name: "Paternity Leave", Code: "PL", MaxDaysPerYear: 5, IsPaid: true, CarryForward: false},
		{Name: "Unpaid Leave", Code: "UL", MaxDaysPerYear: 30, IsPaid: false, CarryForward: false},
		{Name: "Compensatory Off", Code: "CO", MaxDaysPerYear: 12, IsPaid: true, CarryForward: false},
	}
	existing, _ := GetAllLeaveTypes()
	if len(existing) > 0 {
		return nil // already seeded
	}
	for i := range defaults {
		if err := CreateLeaveType(&defaults[i]); err != nil {
			return err
		}
	}
	return nil
}

// AllocateLeaveBalances initialises balance rows for a new employee for the current year.
func AllocateLeaveBalances(userID uint, year int) error {
	leaveTypes, err := GetAllLeaveTypes()
	if err != nil {
		return err
	}
	for _, lt := range leaveTypes {
		b := &LeaveBalance{
			UserID:      userID,
			LeaveTypeID: lt.ID,
			Year:        year,
			Allocated:   float64(lt.MaxDaysPerYear),
		}
		if err := UpsertLeaveBalance(b); err != nil {
			return err
		}
	}
	return nil
}

// ApplyLeave submits a leave request; validates balance and working days.
func ApplyLeave(userID, leaveTypeID uint, startDate, endDate time.Time, halfDay bool, reason string) (*LeaveRequest, error) {
	if endDate.Before(startDate) {
		return nil, errors.New("end date must be after start date")
	}

	totalDays := CountWorkingDays(startDate, endDate)
	if halfDay {
		totalDays = 0.5
	}
	if totalDays == 0 {
		return nil, errors.New("no working days in selected range")
	}

	year := startDate.Year()
	balance, err := GetLeaveBalance(userID, leaveTypeID, year)
	if err != nil {
		return nil, errors.New("leave balance not found for this type")
	}
	if balance.Remaining() < totalDays {
		return nil, errors.New("insufficient leave balance")
	}

	// Reserve in pending
	balance.Pending += totalDays
	if err := SaveLeaveBalance(balance); err != nil {
		return nil, err
	}

	req := &LeaveRequest{
		UserID:      userID,
		LeaveTypeID: leaveTypeID,
		StartDate:   startDate,
		EndDate:     endDate,
		TotalDays:   totalDays,
		HalfDay:     halfDay,
		Reason:      reason,
		Status:      "pending",
	}
	if err := CreateLeaveRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// ApproveLeave moves request from pending → approved and updates balance.
func ApproveLeave(requestID, approverID uint) (*LeaveRequest, error) {
	req, err := GetLeaveRequestByID(requestID)
	if err != nil {
		return nil, errors.New("leave request not found")
	}
	if req.Status != "pending" {
		return nil, errors.New("only pending requests can be approved")
	}

	year := req.StartDate.Year()
	balance, err := GetLeaveBalance(req.UserID, req.LeaveTypeID, year)
	if err != nil {
		return nil, err
	}
	balance.Pending -= req.TotalDays
	balance.Used += req.TotalDays
	if err := SaveLeaveBalance(balance); err != nil {
		return nil, err
	}

	now := time.Now()
	req.Status = "approved"
	req.ApprovedBy = &approverID
	req.ApprovedAt = &now
	if err := UpdateLeaveRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RejectLeave rejects the request and returns days to pending pool.
func RejectLeave(requestID, rejectorID uint, reason string) (*LeaveRequest, error) {
	req, err := GetLeaveRequestByID(requestID)
	if err != nil {
		return nil, errors.New("leave request not found")
	}
	if req.Status != "pending" {
		return nil, errors.New("only pending requests can be rejected")
	}

	year := req.StartDate.Year()
	balance, err := GetLeaveBalance(req.UserID, req.LeaveTypeID, year)
	if err == nil {
		balance.Pending -= req.TotalDays
		_ = SaveLeaveBalance(balance)
	}

	req.Status = "rejected"
	req.RejectedBy = &rejectorID
	req.RejectReason = reason
	if err := UpdateLeaveRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// CancelLeave allows an employee to withdraw a pending request.
func CancelLeave(requestID, userID uint) error {
	req, err := GetLeaveRequestByID(requestID)
	if err != nil {
		return errors.New("leave request not found")
	}
	if req.UserID != userID {
		return errors.New("unauthorized")
	}
	if req.Status != "pending" {
		return errors.New("only pending requests can be cancelled")
	}

	year := req.StartDate.Year()
	balance, err := GetLeaveBalance(req.UserID, req.LeaveTypeID, year)
	if err == nil {
		balance.Pending -= req.TotalDays
		_ = SaveLeaveBalance(balance)
	}

	req.Status = "cancelled"
	return UpdateLeaveRequest(req)
}

// ProcessYearEndCarryForward carries forward eligible balances into next year.
func ProcessYearEndCarryForward(year int) error {
	leaveTypes, err := GetAllLeaveTypes()
	if err != nil {
		return err
	}
	for _, lt := range leaveTypes {
		if !lt.CarryForward {
			continue
		}
		// Get all balances for this type in the ending year
		var balances []LeaveBalance
		balances, err = GetLeaveBalances(0, year) // we'll filter below
		_ = balances
		// Full implementation: iterate all users and carry min(remaining, MaxCarryDays)
		// Placeholder for scheduler hook
	}
	return nil
}
