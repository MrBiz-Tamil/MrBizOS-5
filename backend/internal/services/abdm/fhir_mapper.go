package abdm

import (
	"fmt"
	"time"
)

// FHIRBundle represents a FHIR R4 Bundle resource
type FHIRBundle struct {
	ResourceType string       `json:"resourceType"`
	ID           string       `json:"id"`
	Type         string       `json:"type"` // document, collection, transaction
	Timestamp    string       `json:"timestamp"`
	Entry        []FHIREntry  `json:"entry"`
}

// FHIREntry is a single resource within a Bundle
type FHIREntry struct {
	FullURL  string      `json:"fullUrl"`
	Resource interface{} `json:"resource"`
}

// FHIRPatient represents a FHIR Patient resource
type FHIRPatient struct {
	ResourceType string        `json:"resourceType"`
	ID           string        `json:"id"`
	Identifier   []FHIRIdent   `json:"identifier"`
	Name         []FHIRName    `json:"name"`
	Gender       string        `json:"gender"` // male, female, other, unknown
	BirthDate    string        `json:"birthDate"` // YYYY-MM-DD
	Telecom      []FHIRTelecom `json:"telecom"`
	Address      []FHIRAddress `json:"address,omitempty"`
}

type FHIRIdent struct {
	System string `json:"system"`
	Value  string `json:"value"`
}

type FHIRName struct {
	Use    string   `json:"use"` // official, usual, nickname
	Text   string   `json:"text"`
	Family string   `json:"family"`
	Given  []string `json:"given"`
}

type FHIRTelecom struct {
	System string `json:"system"` // phone, email
	Value  string `json:"value"`
	Use    string `json:"use"` // home, work, mobile
}

type FHIRAddress struct {
	Use        string   `json:"use"`
	Line       []string `json:"line"`
	City       string   `json:"city"`
	State      string   `json:"state"`
	PostalCode string   `json:"postalCode"`
	Country    string   `json:"country"`
}

// FHIRMedication represents a FHIR MedicationRequest resource
type FHIRMedication struct {
	ResourceType     string            `json:"resourceType"`
	ID               string            `json:"id"`
	Status           string            `json:"status"` // active, completed, cancelled
	Intent           string            `json:"intent"` // order, plan, proposal
	MedicationCodeable FHIRCodeable    `json:"medicationCodeableConcept"`
	Subject          FHIRReference     `json:"subject"`
	Requester        FHIRReference     `json:"requester"`
	DosageInstruction []FHIRDosage    `json:"dosageInstruction,omitempty"`
	AuthoredOn       string            `json:"authoredOn"`
}

type FHIRCodeable struct {
	Coding []FHIRCoding `json:"coding"`
	Text   string       `json:"text"`
}

type FHIRCoding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

type FHIRReference struct {
	Reference string `json:"reference"`
	Display   string `json:"display,omitempty"`
}

type FHIRDosage struct {
	Text                string `json:"text"`
	Timing              *FHIRTiming `json:"timing,omitempty"`
	DoseAndRate         []FHIRDoseRate `json:"doseAndRate,omitempty"`
}

type FHIRTiming struct {
	Code FHIRCodeable `json:"code"`
}

type FHIRDoseRate struct {
	DoseQuantity FHIRQuantity `json:"doseQuantity"`
}

type FHIRQuantity struct {
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"`
	System string  `json:"system"`
	Code   string  `json:"code"`
}

// FHIRObservation represents a FHIR Observation (vitals, lab result)
type FHIRObservation struct {
	ResourceType    string       `json:"resourceType"`
	ID              string       `json:"id"`
	Status          string       `json:"status"` // final, preliminary, amended
	Category        []FHIRCodeable `json:"category"`
	Code            FHIRCodeable `json:"code"`
	Subject         FHIRReference `json:"subject"`
	EffectiveDateTime string     `json:"effectiveDateTime"`
	ValueQuantity   *FHIRQuantity `json:"valueQuantity,omitempty"`
	ValueString     string       `json:"valueString,omitempty"`
}

// MapPatientToFHIR converts our internal patient data to FHIR R4 Patient resource
func MapPatientToFHIR(patientID uint, name, gender, dob, mobile, abhaID string) *FHIRPatient {
	fhirGender := "unknown"
	switch gender {
	case "M", "male", "Male":
		fhirGender = "male"
	case "F", "female", "Female":
		fhirGender = "female"
	case "O", "other", "Other":
		fhirGender = "other"
	}

	identifiers := []FHIRIdent{
		{System: "https://Dr.Ayya.in/patient", Value: fmt.Sprintf("%d", patientID)},
	}
	if abhaID != "" {
		identifiers = append(identifiers, FHIRIdent{
			System: "https://healthid.ndhm.gov.in",
			Value:  abhaID,
		})
	}

	return &FHIRPatient{
		ResourceType: "Patient",
		ID:           fmt.Sprintf("patient-%d", patientID),
		Identifier:   identifiers,
		Name: []FHIRName{
			{Use: "official", Text: name},
		},
		Gender:    fhirGender,
		BirthDate: dob,
		Telecom: []FHIRTelecom{
			{System: "phone", Value: mobile, Use: "mobile"},
		},
	}
}

// MapPrescriptionToFHIR converts an internal prescription to FHIR MedicationRequest resources
func MapPrescriptionToFHIR(prescriptionID uint, patientID uint, doctorID uint, medicines []map[string]string) []FHIRMedication {
	result := []FHIRMedication{}
	for i, med := range medicines {
		name := med["name"]
		if name == "" {
			continue
		}
		dosage := med["dosage"]
		freq := med["frequency"]

		result = append(result, FHIRMedication{
			ResourceType: "MedicationRequest",
			ID:           fmt.Sprintf("medreq-%d-%d", prescriptionID, i),
			Status:       "active",
			Intent:       "order",
			MedicationCodeable: FHIRCodeable{
				Text: name,
				Coding: []FHIRCoding{{
					System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
					Display: name,
				}},
			},
			Subject:    FHIRReference{Reference: fmt.Sprintf("Patient/patient-%d", patientID)},
			Requester:  FHIRReference{Reference: fmt.Sprintf("Practitioner/doctor-%d", doctorID)},
			AuthoredOn: time.Now().Format("2006-01-02"),
			DosageInstruction: []FHIRDosage{
				{Text: fmt.Sprintf("%s — %s", dosage, freq)},
			},
		})
	}
	return result
}

// MapVitalToFHIR converts a vital sign reading to FHIR Observation
func MapVitalToFHIR(patientID uint, vitalType, value, unit string) *FHIRObservation {
	loincCodes := map[string]FHIRCoding{
		"blood_pressure":   {System: "http://loinc.org", Code: "55284-4", Display: "Blood pressure systolic and diastolic"},
		"heart_rate":       {System: "http://loinc.org", Code: "8867-4", Display: "Heart rate"},
		"temperature":      {System: "http://loinc.org", Code: "8310-5", Display: "Body temperature"},
		"spo2":             {System: "http://loinc.org", Code: "2708-6", Display: "Oxygen saturation"},
		"respiratory_rate": {System: "http://loinc.org", Code: "9279-1", Display: "Respiratory rate"},
		"weight":           {System: "http://loinc.org", Code: "29463-7", Display: "Body weight"},
		"height":           {System: "http://loinc.org", Code: "8302-2", Display: "Body height"},
		"bmi":              {System: "http://loinc.org", Code: "39156-5", Display: "Body mass index"},
	}

	coding := loincCodes[vitalType]
	if coding.Code == "" {
		coding = FHIRCoding{System: "http://Dr.Ayya.in/vitals", Code: vitalType, Display: vitalType}
	}

	return &FHIRObservation{
		ResourceType: "Observation",
		ID:           fmt.Sprintf("obs-%d-%s-%d", patientID, vitalType, time.Now().Unix()),
		Status:       "final",
		Category: []FHIRCodeable{{
			Coding: []FHIRCoding{{System: "http://terminology.hl7.org/CodeSystem/observation-category", Code: "vital-signs"}},
		}},
		Code:              FHIRCodeable{Coding: []FHIRCoding{coding}, Text: coding.Display},
		Subject:           FHIRReference{Reference: fmt.Sprintf("Patient/patient-%d", patientID)},
		EffectiveDateTime: time.Now().Format(time.RFC3339),
		ValueString:       fmt.Sprintf("%s %s", value, unit),
	}
}

// BuildPatientBundle creates a FHIR Bundle document for a patient's health records
func BuildPatientBundle(patient *FHIRPatient, medications []FHIRMedication, vitals []FHIRObservation) *FHIRBundle {
	bundle := &FHIRBundle{
		ResourceType: "Bundle",
		ID:           fmt.Sprintf("bundle-%d", time.Now().Unix()),
		Type:         "document",
		Timestamp:    time.Now().Format(time.RFC3339),
		Entry:        []FHIREntry{},
	}

	// Add patient
	bundle.Entry = append(bundle.Entry, FHIREntry{
		FullURL:  fmt.Sprintf("urn:uuid:%s", patient.ID),
		Resource: patient,
	})

	// Add medications
	for _, med := range medications {
		bundle.Entry = append(bundle.Entry, FHIREntry{
			FullURL:  fmt.Sprintf("urn:uuid:%s", med.ID),
			Resource: med,
		})
	}

	// Add vitals
	for _, obs := range vitals {
		bundle.Entry = append(bundle.Entry, FHIREntry{
			FullURL:  fmt.Sprintf("urn:uuid:%s", obs.ID),
			Resource: obs,
		})
	}

	return bundle
}
