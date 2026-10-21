package healthcare

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	healthcarev1 "drayya-core/gen/healthcare/v1"
	"drayya-core/gen/healthcare/v1/healthcarev1connect"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ListDoctors(ctx context.Context, req *connect.Request[healthcarev1.ListDoctorsRequest]) (*connect.Response[healthcarev1.ListDoctorsResponse], error) {
	doctors := []*healthcarev1.DoctorProto{
		{
			Id:                 "doc-1",
			TenantId:           "tenant-1",
			Name:               "Dr. Priya Sharma",
			Specialty:          "Cardiologist",
			Qualification:      "MD, DM (Cardiology)",
			RegistrationNumber: "MCI-48291",
			ExperienceYears:    12,
			ConsultationFee:    "800.00",
			IsAvailable:        true,
		},
		{
			Id:                 "doc-2",
			TenantId:           "tenant-1",
			Name:               "Dr. Arun Verma",
			Specialty:          "General Physician",
			Qualification:      "MBBS, MD",
			RegistrationNumber: "MCI-92812",
			ExperienceYears:    8,
			ConsultationFee:    "500.00",
			IsAvailable:        true,
		},
	}
	return connect.NewResponse(&healthcarev1.ListDoctorsResponse{Doctors: doctors}), nil
}

func (s *Server) GetDoctor(ctx context.Context, req *connect.Request[healthcarev1.GetDoctorRequest]) (*connect.Response[healthcarev1.GetDoctorResponse], error) {
	return connect.NewResponse(&healthcarev1.GetDoctorResponse{
		Doctor: &healthcarev1.DoctorProto{
			Id:                 req.Msg.Id,
			TenantId:           "tenant-1",
			Name:               "Dr. Priya Sharma",
			Specialty:          "Cardiologist",
			Qualification:      "MD, DM",
			RegistrationNumber: "MCI-48291",
			ExperienceYears:    12,
			ConsultationFee:    "800.00",
			IsAvailable:        true,
		},
	}), nil
}

func (s *Server) ListAppointments(ctx context.Context, req *connect.Request[healthcarev1.ListAppointmentsRequest]) (*connect.Response[healthcarev1.ListAppointmentsResponse], error) {
	appts := []*healthcarev1.AppointmentProto{
		{
			Id:               "apt-101",
			TenantId:         "tenant-1",
			DoctorId:         "doc-1",
			DoctorName:       "Dr. Priya Sharma",
			PatientId:        "pat-201",
			PatientName:      "Rajesh Kumar",
			ScheduledAt:      time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			Status:           "CONFIRMED",
			ConsultationType: "VIDEO",
			RoomToken:        "livekit-room-token-mock-xyz",
			Notes:            "Follow-up checkup",
		},
	}
	return connect.NewResponse(&healthcarev1.ListAppointmentsResponse{Appointments: appts}), nil
}

func (s *Server) BookAppointment(ctx context.Context, req *connect.Request[healthcarev1.BookAppointmentRequest]) (*connect.Response[healthcarev1.BookAppointmentResponse], error) {
	appt := &healthcarev1.AppointmentProto{
		Id:               fmt.Sprintf("apt-%d", time.Now().Unix()),
		TenantId:         "tenant-1",
		DoctorId:         req.Msg.DoctorId,
		DoctorName:       "Dr. Priya Sharma",
		PatientName:      req.Msg.PatientName,
		ScheduledAt:      req.Msg.ScheduledAt,
		Status:           "CONFIRMED",
		ConsultationType: req.Msg.ConsultationType,
		RoomToken:        "livekit-room-token-generated",
		Notes:            req.Msg.Notes,
	}
	return connect.NewResponse(&healthcarev1.BookAppointmentResponse{Appointment: appt}), nil
}

func (s *Server) UpdateAppointmentStatus(ctx context.Context, req *connect.Request[healthcarev1.UpdateAppointmentStatusRequest]) (*connect.Response[healthcarev1.UpdateAppointmentStatusResponse], error) {
	return connect.NewResponse(&healthcarev1.UpdateAppointmentStatusResponse{
		Appointment: &healthcarev1.AppointmentProto{
			Id:     req.Msg.Id,
			Status: req.Msg.Status,
		},
	}), nil
}

func (s *Server) CreatePrescription(ctx context.Context, req *connect.Request[healthcarev1.CreatePrescriptionRequest]) (*connect.Response[healthcarev1.CreatePrescriptionResponse], error) {
	rx := &healthcarev1.PrescriptionProto{
		Id:            fmt.Sprintf("rx-%d", time.Now().Unix()),
		TenantId:      "tenant-1",
		AppointmentId: req.Msg.AppointmentId,
		PatientId:     req.Msg.PatientId,
		Diagnosis:     req.Msg.Diagnosis,
		Items:         req.Msg.Items,
		Advice:        req.Msg.Advice,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}
	return connect.NewResponse(&healthcarev1.CreatePrescriptionResponse{Prescription: rx}), nil
}

func (s *Server) GetPatientMedicalRecord(ctx context.Context, req *connect.Request[healthcarev1.GetPatientMedicalRecordRequest]) (*connect.Response[healthcarev1.GetPatientMedicalRecordResponse], error) {
	return connect.NewResponse(&healthcarev1.GetPatientMedicalRecordResponse{
		PatientId: req.Msg.PatientId,
	}), nil
}

func (s *Server) StreamConsultationEvents(ctx context.Context, req *connect.Request[healthcarev1.StreamConsultationEventsRequest], stream *connect.ServerStream[healthcarev1.ConsultationEventProto]) error {
	// Sample event streaming
	return stream.Send(&healthcarev1.ConsultationEventProto{
		EventType:   "CALL_CONNECTED",
		PayloadJson: fmt.Sprintf(`{"roomId":"%s","status":"active"}`, req.Msg.RoomId),
		Timestamp:   time.Now().Unix(),
	})
}

// Compile-time check
var _ healthcarev1connect.HealthcareServiceHandler = (*Server)(nil)
