package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const BaseURL = "http://localhost:8081/api"

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResp struct {
	Token string      `json:"access_token"`
	User  interface{} `json:"user"`
}

type AppointmentReq struct {
	PatientID        uint   `json:"patient_id"`
	DoctorID         *uint  `json:"doctor_id"`
	AppointmentType  string `json:"appointment_type"`
	ConsultationMode string `json:"consultation_mode"`
	Notes            string `json:"notes"`
}

// Minimal struct to capture ID
type AppointmentResp struct {
	ID        uint   `json:"id"`
	PatientID uint   `json:"patient_id"`
	Status    string `json:"status"`
}

func main() {
	log.Println("     Starting Live Integration Test (Port 8081)")

	// 1. Login Patient
	log.Println("     Logging in Patient...")
	patToken, patID := login("patient@Dr.Ayya.com", "patient123")
	log.Printf("    Patient Logged In (ID: %d)", patID)

	// Topup Patient Wallet
	log.Println("     Topping up Patient Wallet...")
	topupWallet(patToken, 5000)

	// 2. Login Doctor
	log.Println("              Logging in Doctor...")
	docToken, docID := login("doctor@Dr.Ayya.com", "doctor123")
	log.Printf("    Doctor Logged In (ID: %d)", docID)

	// 3. Create Instant Appointment
	log.Println("     Creating Instant Appointment...")
	appt := createAppointment(patToken, patID)
	log.Printf("    Appointment Created (ID: %d, Status: %s)", appt.ID, appt.Status)

	// 4. Force Assign & Accept (Simulating Dispatcher/Doctor)
	// Since we don't have a direct "Force Assign" API exposed usually, we use the Doctor's token to "Update" if possible,
	// or we rely on the manual assign endpoint if available.
	// Let's try to update the appointment notes or status if allowed, or use special dispatch endpoint.

	// Actually, we can use the "Manual Assign" dispatch endpoint if we are admin.
	// Or we can just use the DB directly? No, this is a live test against HTTP.

	// Let's try to "Accept" it directly as the doctor.
	// BUT, the appointment needs to be assigned first.
	// If the dispatcher is running, it might have auto-assigned.
	// Let's poll for assignment.

	log.Println("    Waiting for Dispatcher Assignment...")
	for i := 0; i < 10; i++ {
		updatedAppt := getAppointment(patToken, appt.ID)
		// Check if doctor_id is set in the raw response map (not shown in minimal struct, let's fix struct)
		// Actually, let's just assume if status changes or check /dispatch/queue
		if updatedAppt.Status != "pending" {
			log.Printf("    Appointment Updated: Status=%s", updatedAppt.Status)
			break
		}
		time.Sleep(1 * time.Second)
	}

	// If still pending, try manual assign (Assuming we have admin token or just try doctor accepting SOS?)
	// Regular appointments might not allow "Doctor Accept" if not assigned.
	// Let's try the SOS "Accept" endpoint? No, this is "Instant".

	// Let's use the dispatcher manual assign endpoint.
	// Needs Admin.
	log.Println("     Logging in Admin...")
	adminToken, _ := login("admin@Dr.Ayya.com", "admin123")

	log.Println("        Manually Assigning Doctor...")
	assignReq := map[string]interface{}{
		"appointment_id": appt.ID,
		"doctor_id":      docID,
		"reason":         "Integration Test Force Assign",
	}
	status := postRequest(adminToken, "/dispatch/manual/assign", assignReq)
	if status != 200 && status != 201 {
		log.Fatalf("    Failed to assign doctor. Status: %d", status)
	}
	log.Println("    Doctor Assigned via Dispatch Endpoint")

	// 5. Doctor Accepts (Update Status)
	// The previous step might set it to "assigned" or "confirmed".
	// Let's have the doctor mark it "in_progress" (Start Consultation).
	// Endpoint: PUT /appointments/:id (UpdateAppointment)
	log.Println("              Doctor Starting Consultation...")
	updateReq := map[string]interface{}{
		"status": "in_progress",
	}
	// Note: UpdateAppointment handler usually allows doctor to update status
	// Or we use /livekit/call/start
	status = putRequest(docToken, fmt.Sprintf("/appointments/%d", appt.ID), updateReq)
	if status != 200 {
		log.Printf("       Failed to update status directly (Status: %d). Trying Call Start...", status)
		// Try Start Call
		callReq := map[string]interface{}{
			"appointment_id": appt.ID,
		}
		status = postRequest(docToken, "/livekit/call/start", callReq)
		if status != 200 {
			log.Fatalf("    Failed to start call. Status: %d", status)
		}
		log.Println("    Call Started (LiveKit)")
	} else {
		log.Println("    Consultation Started (Status: in_progress)")
	}

	log.Println("     Live Integration Test Completed Successfully!")
	log.Println("        Go to http://localhost:3000 to see the active appointment.")
}

func login(email, password string) (string, uint) {
	req := LoginReq{Email: email, Password: password}
	respBody, status := makeRequest("POST", "/auth/login", "", req)
	if status != 200 {
		log.Fatalf("Login failed for %s. Status: %d, Body: %s", email, status, respBody)
	}

	var resp LoginResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		log.Fatalf("Failed to unmarshal login response: %v. Body: %s", err, respBody)
	}

	if resp.Token == "" {
		log.Fatalf("Login response token is empty. Body: %s", respBody)
	}

	log.Printf("Got Token for %s: %s...", email, resp.Token[:10])

	// Extract ID map
	userMap, ok := resp.User.(map[string]interface{})
	if !ok {
		log.Fatalf("User field is not a map. Body: %s", respBody)
	}

	idFloat, ok := userMap["id"].(float64)
	if !ok {
		log.Fatalf("User ID not found or not float. UserMap: %v", userMap)
	}

	return resp.Token, uint(idFloat)
}

func topupWallet(token string, amount int) {
	req := map[string]interface{}{"amount": amount}
	_, status := makeRequest("POST", "/wallet/topup", token, req)
	if status != 200 {
		log.Fatalf("Topup failed. Status: %d", status)
	}
}

func createAppointment(token string, patientID uint) AppointmentResp {
	req := AppointmentReq{
		PatientID:        patientID,
		AppointmentType:  "instant",
		ConsultationMode: "video",
		Notes:            "Live Test Integration",
	}
	respBody, status := makeRequest("POST", "/appointments/", token, req)
	if status != 201 {
		log.Fatalf("Create Appointment failed. Status: %d, Body: %s", status, respBody)
	}

	var appt AppointmentResp
	json.Unmarshal(respBody, &appt)
	return appt
}

func getAppointment(token string, id uint) AppointmentResp {
	respBody, status := makeRequest("GET", fmt.Sprintf("/appointments/%d", id), token, nil)
	if status != 200 {
		log.Fatalf("Get Appointment failed. Status: %d", status)
	}
	var appt AppointmentResp
	json.Unmarshal(respBody, &appt)
	return appt
}

func postRequest(token, endpoint string, body interface{}) int {
	_, status := makeRequest("POST", endpoint, token, body)
	return status
}

func putRequest(token, endpoint string, body interface{}) int {
	_, status := makeRequest("PUT", endpoint, token, body)
	return status
}

func makeRequest(method, endpoint, token string, body interface{}) ([]byte, int) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBytes)
	}

	req, _ := http.NewRequest(method, BaseURL+endpoint, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		// Debug: Log the header (masked)
		// log.Printf("Sending Auth: Bearer %s...", token[:5])
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		log.Printf("       401 Unauthorized for %s %s", method, endpoint)
		log.Printf("Token Length: %d", len(token))
		log.Printf("Response Body: %s", string(respBody))
	}

	return respBody, resp.StatusCode
}

