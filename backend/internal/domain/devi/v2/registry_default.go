package v2

// DefaultRegistry builds the registry of tools shipped in Phase 0. Each
// phase adds more tools; this is the central place to wire them in.
//
// Convention: each tool lives in tool_<name>.go with a public *Tool variable
// named <name>Tool. Register here.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	// Phase 0 — first two read tools
	r.Register(searchDoctorsTool)
	r.Register(lookupKnowledgeTool)
	// Phase 2 — five more read tools
	r.Register(searchProductsTool)
	r.Register(getDoctorAvailabilityTool)
	r.Register(getMyPrescriptionsTool)
	r.Register(findSimilarSymptomsTool)
	r.Register(switchFamilyContextTool)
	// Phase 4 — write tools
	r.Register(addToCartTool)
	r.Register(updatePatientProfileTool)
	r.Register(createBookingTool)
	return r
}
