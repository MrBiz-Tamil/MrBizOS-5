package healthcare

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"
	healthcarev1 "drayya-core/gen/healthcare/v1"
	"drayya-core/gen/healthcare/v1/healthcarev1connect"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/identity"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ListDoctors(ctx context.Context, req *connect.Request[healthcarev1.ListDoctorsRequest]) (*connect.Response[healthcarev1.ListDoctorsResponse], error) {
	var doctors []*healthcarev1.DoctorProto
	if s.db != nil {
		var profiles []identity.DoctorProfile
		if err := s.db.WithContext(ctx).Find(&profiles).Error; err == nil && len(profiles) > 0 {
			userMap := make(map[uint]identity.User)
			var userIDs []uint
			for _, p := range profiles {
				userIDs = append(userIDs, p.UserID)
			}
			if len(userIDs) > 0 {
				var users []identity.User
				if err := s.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err == nil {
					for _, u := range users {
						userMap[u.ID] = u
					}
				}
			}

			for _, p := range profiles {
				name := "Dr. Specialist"
				if u, ok := userMap[p.UserID]; ok && u.FullName != "" {
					name = u.FullName
				}
				fee := "500.00"
				if p.ConsultationFees != nil && p.ConsultationFees["INR"] > 0 {
					fee = fmt.Sprintf("%.2f", float64(p.ConsultationFees["INR"])/100.0)
				}
				doctors = append(doctors, &healthcarev1.DoctorProto{
					Id:                 strconv.FormatUint(uint64(p.ID), 10),
					TenantId:           "tenant-1",
					Name:               name,
					Specialty:          p.Specialization,
					Qualification:      p.Qualification,
					RegistrationNumber: p.LicenseNumber,
					ExperienceYears:    int32(p.YearsOfExperience),
					ConsultationFee:    fee,
					IsAvailable:        p.IsOnline,
				})
			}
		}
	}

	if len(doctors) == 0 {
		doctors = []*healthcarev1.DoctorProto{
			{
				Id:                 "1",
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
				Id:                 "2",
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
	}
	return connect.NewResponse(&healthcarev1.ListDoctorsResponse{Doctors: doctors}), nil
}

func (s *Server) GetDoctor(ctx context.Context, req *connect.Request[healthcarev1.GetDoctorRequest]) (*connect.Response[healthcarev1.GetDoctorResponse], error) {
	if s.db != nil {
		var p identity.DoctorProfile
		if err := s.db.WithContext(ctx).Where("id = ? OR user_id = ?", req.Msg.Id, req.Msg.Id).First(&p).Error; err == nil {
			name := "Dr. Specialist"
			var u identity.User
			if err := s.db.WithContext(ctx).First(&u, p.UserID).Error; err == nil && u.FullName != "" {
				name = u.FullName
			}
			fee := "500.00"
			if p.ConsultationFees != nil && p.ConsultationFees["INR"] > 0 {
				fee = fmt.Sprintf("%.2f", float64(p.ConsultationFees["INR"])/100.0)
			}
			return connect.NewResponse(&healthcarev1.GetDoctorResponse{
				Doctor: &healthcarev1.DoctorProto{
					Id:                 strconv.FormatUint(uint64(p.ID), 10),
					TenantId:           "tenant-1",
					Name:               name,
					Specialty:          p.Specialization,
					Qualification:      p.Qualification,
					RegistrationNumber: p.LicenseNumber,
					ExperienceYears:    int32(p.YearsOfExperience),
					ConsultationFee:    fee,
					IsAvailable:        p.IsOnline,
				},
			}), nil
		}
	}
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
	var appts []*healthcarev1.AppointmentProto
	if s.db != nil {
		var rows []appointment.Appointment
		if err := s.db.WithContext(ctx).Order("start_time DESC").Find(&rows).Error; err == nil && len(rows) > 0 {
			userMap := make(map[uint]identity.User)
			var userIDs []uint
			for _, r := range rows {
				userIDs = append(userIDs, r.PatientID)
				if r.DoctorID != nil {
					userIDs = append(userIDs, *r.DoctorID)
				}
			}
			if len(userIDs) > 0 {
				var users []identity.User
				if err := s.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err == nil {
					for _, u := range users {
						userMap[u.ID] = u
					}
				}
			}

			for _, r := range rows {
				docName := "Consulting Doctor"
				if r.DoctorID != nil {
					if u, ok := userMap[*r.DoctorID]; ok && u.FullName != "" {
						docName = u.FullName
					}
				}
				patName := "Patient"
				if u, ok := userMap[r.PatientID]; ok && u.FullName != "" {
					patName = u.FullName
				}
				docIDStr := ""
				if r.DoctorID != nil {
					docIDStr = strconv.FormatUint(uint64(*r.DoctorID), 10)
				}
				appts = append(appts, &healthcarev1.AppointmentProto{
					Id:               strconv.FormatUint(uint64(r.ID), 10),
					TenantId:         "tenant-1",
					DoctorId:         docIDStr,
					DoctorName:       docName,
					PatientId:        strconv.FormatUint(uint64(r.PatientID), 10),
					PatientName:      patName,
					ScheduledAt:      r.StartTime.Format(time.RFC3339),
					Status:           string(r.Status),
					ConsultationType: r.ConsultationMode,
					RoomToken:        fmt.Sprintf("livekit-room-%d", r.ID),
					Notes:            r.Notes,
				})
			}
		}
	}

	if len(appts) == 0 {
		appts = []*healthcarev1.AppointmentProto{
			{
				Id:               "101",
				TenantId:         "tenant-1",
				DoctorId:         "1",
				DoctorName:       "Dr. Priya Sharma",
				PatientId:        "201",
				PatientName:      "Rajesh Kumar",
				ScheduledAt:      time.Now().Add(2 * time.Hour).Format(time.RFC3339),
				Status:           "CONFIRMED",
				ConsultationType: "VIDEO",
				RoomToken:        "livekit-room-token-live",
				Notes:            "Follow-up consultation",
			},
		}
	}
	return connect.NewResponse(&healthcarev1.ListAppointmentsResponse{Appointments: appts}), nil
}

func (s *Server) BookAppointment(ctx context.Context, req *connect.Request[healthcarev1.BookAppointmentRequest]) (*connect.Response[healthcarev1.BookAppointmentResponse], error) {
	docID, _ := strconv.ParseUint(req.Msg.DoctorId, 10, 64)
	docIDUint := uint(docID)

	var startTime time.Time
	if t, err := time.Parse(time.RFC3339, req.Msg.ScheduledAt); err == nil {
		startTime = t
	} else {
		startTime = time.Now().Add(1 * time.Hour)
	}

	newAppt := &appointment.Appointment{
		DoctorID:         &docIDUint,
		StartTime:        startTime,
		EndTime:          startTime.Add(30 * time.Minute),
		Status:           appointment.StatusConfirmed,
		AppointmentType:  "consultation",
		ConsultationMode: req.Msg.ConsultationType,
		Notes:            req.Msg.Notes,
		ConsultationFee:  50000,
	}

	if s.db != nil {
		s.db.WithContext(ctx).Create(newAppt)
	}

	appt := &healthcarev1.AppointmentProto{
		Id:               strconv.FormatUint(uint64(newAppt.ID), 10),
		TenantId:         "tenant-1",
		DoctorId:         req.Msg.DoctorId,
		DoctorName:       "Consulting Doctor",
		PatientName:      req.Msg.PatientName,
		ScheduledAt:      newAppt.StartTime.Format(time.RFC3339),
		Status:           string(newAppt.Status),
		ConsultationType: req.Msg.ConsultationType,
		RoomToken:        fmt.Sprintf("livekit-room-%d", newAppt.ID),
		Notes:            req.Msg.Notes,
	}
	return connect.NewResponse(&healthcarev1.BookAppointmentResponse{Appointment: appt}), nil
}

func (s *Server) UpdateAppointmentStatus(ctx context.Context, req *connect.Request[healthcarev1.UpdateAppointmentStatusRequest]) (*connect.Response[healthcarev1.UpdateAppointmentStatusResponse], error) {
	if s.db != nil {
		s.db.WithContext(ctx).Model(&appointment.Appointment{}).Where("id = ?", req.Msg.Id).Update("status", req.Msg.Status)
	}
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
	return stream.Send(&healthcarev1.ConsultationEventProto{
		EventType:   "CALL_CONNECTED",
		PayloadJson: fmt.Sprintf(`{"roomId":"%s","status":"active"}`, req.Msg.RoomId),
		Timestamp:   time.Now().Unix(),
	})
}

// Compile-time check
var _ healthcarev1connect.HealthcareServiceHandler = (*Server)(nil)
