package emr

// Prescription route handling is implemented in the domain handler:
// internal/domain/emr/handler.go — registered via emr.NewHandler(emrSvc, db).RegisterRoutes(v1)
//
// Endpoints available under /api/v1/emr/prescriptions:
//   POST   /prescriptions                            — create prescription (doctor only)
//   GET    /prescriptions                            — list prescriptions issued by the authenticated doctor
//   GET    /prescriptions/my                         — list prescriptions received by the authenticated patient
//   GET    /prescriptions/appointment/:appointment_id — get prescription for a specific appointment
//   GET    /prescriptions/patient/:patient_id        — list prescriptions for a patient (admin/doctor)
//   GET    /prescriptions/:id                        — get a single prescription by ID
//   GET    /prescriptions/:id/download               — stream the prescription PDF
