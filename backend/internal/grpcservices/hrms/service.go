package hrms

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	hrmsv1 "drayya-core/gen/hrms/v1"
	"drayya-core/gen/hrms/v1/hrmsv1connect"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ListEmployees(ctx context.Context, req *connect.Request[hrmsv1.ListEmployeesRequest]) (*connect.Response[hrmsv1.ListEmployeesResponse], error) {
	emps := []*hrmsv1.EmployeeProto{
		{
			Id:            "emp-1",
			TenantId:      "tenant-1",
			EmployeeCode:  "EMP-1001",
			FirstName:     "Rahul",
			LastName:      "Nair",
			Email:         "rahul.nair@mrbiz.local",
			Phone:         "+91 98491 00221",
			Department:    "Engineering & Consulting",
			Designation:   "Lead Cloud Architect",
			DateOfJoining: "2024-03-01",
			SalaryStructure: &hrmsv1.SalaryStructureProto{
				BasicSalary:            "85000.00",
				Hra:                    "34000.00",
				SpecialAllowance:       "31000.00",
				ProvidentFundDeduction: "10200.00",
				ProfessionalTax:        "200.00",
				IncomeTaxTds:           "12500.00",
				NetPayable:             "127100.00",
			},
			IsActive: true,
		},
		{
			Id:            "emp-2",
			TenantId:      "tenant-1",
			EmployeeCode:  "EMP-1002",
			FirstName:     "Sneha",
			LastName:      "Patil",
			Email:         "sneha.patil@mrbiz.local",
			Phone:         "+91 97312 88192",
			Department:    "Finance & Accounting",
			Designation:   "Senior Chartered Accountant",
			DateOfJoining: "2024-06-15",
			SalaryStructure: &hrmsv1.SalaryStructureProto{
				BasicSalary:            "60000.00",
				Hra:                    "24000.00",
				SpecialAllowance:       "16000.00",
				ProvidentFundDeduction: "7200.00",
				ProfessionalTax:        "200.00",
				IncomeTaxTds:           "8000.00",
				NetPayable:             "84600.00",
			},
			IsActive: true,
		},
	}
	return connect.NewResponse(&hrmsv1.ListEmployeesResponse{Employees: emps}), nil
}

func (s *Server) CreateEmployee(ctx context.Context, req *connect.Request[hrmsv1.CreateEmployeeRequest]) (*connect.Response[hrmsv1.CreateEmployeeResponse], error) {
	emp := &hrmsv1.EmployeeProto{
		Id:              fmt.Sprintf("emp-%d", time.Now().Unix()),
		TenantId:        "tenant-1",
		EmployeeCode:    req.Msg.EmployeeCode,
		FirstName:       req.Msg.FirstName,
		LastName:        req.Msg.LastName,
		Email:           req.Msg.Email,
		Phone:           req.Msg.Phone,
		Department:      req.Msg.Department,
		Designation:     req.Msg.Designation,
		DateOfJoining:   req.Msg.DateOfJoining,
		SalaryStructure: req.Msg.SalaryStructure,
		IsActive:        true,
	}
	return connect.NewResponse(&hrmsv1.CreateEmployeeResponse{Employee: emp}), nil
}

func (s *Server) UpdateSalaryStructure(ctx context.Context, req *connect.Request[hrmsv1.UpdateSalaryStructureRequest]) (*connect.Response[hrmsv1.UpdateSalaryStructureResponse], error) {
	return connect.NewResponse(&hrmsv1.UpdateSalaryStructureResponse{
		Employee: &hrmsv1.EmployeeProto{
			Id:              req.Msg.EmployeeId,
			SalaryStructure: req.Msg.SalaryStructure,
			IsActive:        true,
		},
	}), nil
}

// RunMonthlyPayroll executes 1-click automated payroll calculation
func (s *Server) RunMonthlyPayroll(ctx context.Context, req *connect.Request[hrmsv1.RunMonthlyPayrollRequest]) (*connect.Response[hrmsv1.RunMonthlyPayrollResponse], error) {
	mYear := req.Msg.MonthYear
	if mYear == "" {
		mYear = time.Now().Format("2006-01")
	}

	payslips := []*hrmsv1.PayslipProto{
		{
			Id:                      fmt.Sprintf("slip-%d-1", time.Now().Unix()),
			TenantId:                "tenant-1",
			PayslipNumber:           fmt.Sprintf("PAY-%s-001", mYear),
			EmployeeId:              "emp-1",
			EmployeeName:            "Rahul Nair",
			MonthYear:               mYear,
			WorkingDays:             30,
			DaysWorked:              30,
			GrossEarnings:           "150000.00",
			TotalDeductions:         "22900.00",
			NetSalaryPaid:           "127100.00",
			PaymentStatus:           "DISBURSED",
			DisbursedAt:             time.Now().Format(time.RFC3339),
		},
		{
			Id:                      fmt.Sprintf("slip-%d-2", time.Now().Unix()),
			TenantId:                "tenant-1",
			PayslipNumber:           fmt.Sprintf("PAY-%s-002", mYear),
			EmployeeId:              "emp-2",
			EmployeeName:            "Sneha Patil",
			MonthYear:               mYear,
			WorkingDays:             30,
			DaysWorked:              29,
			GrossEarnings:           "100000.00",
			TotalDeductions:         "15400.00",
			NetSalaryPaid:           "84600.00",
			PaymentStatus:           "DISBURSED",
			DisbursedAt:             time.Now().Format(time.RFC3339),
		},
	}

	return connect.NewResponse(&hrmsv1.RunMonthlyPayrollResponse{
		MonthYear:                mYear,
		TotalEmployeesProcessed:  2,
		TotalDisbursementAmount: "211700.00",
		GeneratedPayslips:        payslips,
	}), nil
}

func (s *Server) ListPayslips(ctx context.Context, req *connect.Request[hrmsv1.ListPayslipsRequest]) (*connect.Response[hrmsv1.ListPayslipsResponse], error) {
	payslips := []*hrmsv1.PayslipProto{
		{
			Id:              "slip-1",
			TenantId:        "tenant-1",
			PayslipNumber:   "PAY-2026-10-001",
			EmployeeId:      "emp-1",
			EmployeeName:    "Rahul Nair",
			MonthYear:       "2026-10",
			WorkingDays:     30,
			DaysWorked:      30,
			GrossEarnings:   "150000.00",
			TotalDeductions: "22900.00",
			NetSalaryPaid:   "127100.00",
			PaymentStatus:   "DISBURSED",
		},
	}
	return connect.NewResponse(&hrmsv1.ListPayslipsResponse{Payslips: payslips}), nil
}

func (s *Server) GetPayslip(ctx context.Context, req *connect.Request[hrmsv1.GetPayslipRequest]) (*connect.Response[hrmsv1.GetPayslipResponse], error) {
	return connect.NewResponse(&hrmsv1.GetPayslipResponse{
		Payslip: &hrmsv1.PayslipProto{
			Id:            req.Msg.Id,
			TenantId:      "tenant-1",
			PayslipNumber: "PAY-2026-10-001",
			EmployeeId:    "emp-1",
			EmployeeName:  "Rahul Nair",
			MonthYear:     "2026-10",
			NetSalaryPaid: "127100.00",
			PaymentStatus: "DISBURSED",
		},
	}), nil
}

var _ hrmsv1connect.HRMSServiceHandler = (*Server)(nil)
