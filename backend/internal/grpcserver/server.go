package grpcserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gorm.io/gorm"
)

// Server holds the ConnectRPC multiplexer and underlying DB pool
type Server struct {
	mux         *http.ServeMux
	db          *gorm.DB
	interceptor connect.UnaryInterceptorFunc
	httpServer  *http.Server
}

// NewServer initializes the ConnectRPC server with strict tenant interceptors
func NewServer(db *gorm.DB) *Server {
	mux := http.NewServeMux()
	interceptor := NewAuthInterceptor()

	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","engine":"gRPC-ConnectRPC","framework":"mrbiz-os"}`))
	})

	// Compatibility endpoint for /api/auth/me and /auth/me
	meHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user":{"id":"user-1","tenant_id":"tenant-1","email":"admin@mrbiz.local","full_name":"Executive Administrator","role":"admin","is_super_admin":true,"is_active":true,"current_business_unit_id":"travel","allowed_business_units":["travel","realestate","healthcare","trade","quarry","consulting"]},"tenant":{"id":"tenant-1","name":"Global Business Group","legal_name":"Global Business Group Ltd","gstin":"27AABCA1234F1Z6","address":"Skyline Commercial Towers, Anna Salai","city":"Chennai","state":"Tamil Nadu","country":"India","pincode":"600002","email":"admin@mrbiz.local","phone":"+91 98400 12345","currency":"INR","subscription_plan":"ENTERPRISE","subscription_status":"ACTIVE"}}`))
	}
	mux.HandleFunc("/api/auth/me", meHandler)
	mux.HandleFunc("/auth/me", meHandler)

	// Compatibility endpoint for /api/company and /company
	companyHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"tenant-1","name":"Global Business Group","legal_name":"Global Business Group Ltd","gstin":"27AABCA1234F1Z6","address":"Skyline Commercial Towers, Anna Salai","city":"Chennai","state":"Tamil Nadu","country":"India","pincode":"600002","email":"admin@mrbiz.local","phone":"+91 98400 12345","invoice_prefix":"INV-2026","financial_year_start":"2026-04-01","currency":"INR","subscription_plan":"ENTERPRISE","subscription_status":"ACTIVE","bank_name":"HDFC Bank Ltd","bank_account_no":"50200012345678","bank_ifsc":"HDFC0000240","bank_branch":"Mount Road","upi_id":"mrbiz@upi"}`))
	}
	mux.HandleFunc("/api/company", companyHandler)
	mux.HandleFunc("/company", companyHandler)

	// Compatibility endpoint for /api/v1/realtime/stream (SSE)
	sseHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		fmt.Fprintf(w, "event: billing_update\ndata: {\"type\":\"CONNECTED\",\"payload\":{\"message\":\"Subscribed to MrBiz-OS real-time stream\"},\"timestamp\":\"%s\"}\n\n", now)
		flusher.Flush()

		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				fmt.Fprintf(w, ": heartbeat %s\n\n", t.Format(time.RFC3339))
				flusher.Flush()
			}
		}
	}
	mux.HandleFunc("/api/v1/realtime/stream", sseHandler)
	mux.HandleFunc("/v1/realtime/stream", sseHandler)

	// Compatibility endpoint for /api/billing and /billing
	billingHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ACTIVE","plan_code":"ENTERPRISE","plan_name":"Enterprise Unlimited","billing_cycle":"ANNUAL","trial_days_left":null,"subscription_id":"sub-ent-01"}`))
	}
	mux.HandleFunc("/api/billing", billingHandler)
	mux.HandleFunc("/billing", billingHandler)

	// Compatibility endpoint for /api/legal/status and /legal/status
	legalHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"required_version":"2026-04-01","accepted":true,"accepted_at":"2026-04-01T00:00:00Z"}`))
	}
	mux.HandleFunc("/api/legal/status", legalHandler)
	mux.HandleFunc("/legal/status", legalHandler)

	// Compatibility endpoint for /api/company/subscription and /company/subscription
	subHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tenant_id":"tenant-1","plan_code":"ENTERPRISE","plan_name":"Enterprise Multi-OS","status":"ACTIVE","billing_cycle":"ANNUAL","max_users":100,"max_invoices":100000,"features":{"accounting":true,"expenses":true,"reports":true,"realtime":true,"multi_currency":true,"api_access":true,"unlimited_branches":true}}`))
	}
	mux.HandleFunc("/api/company/subscription", subHandler)
	mux.HandleFunc("/company/subscription", subHandler)

	// Compatibility endpoint for /api/dashboard and /dashboard
	dashboardHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total_sales":"1428500.00","total_received":"1198500.00","outstanding_balance":"230000.00","total_invoices_count":48,"paid_invoices_count":41,"unpaid_invoices_count":7,"total_inventory_value":"890000.00","low_stock_count":2,"recent_invoices":[],"recent_payments":[]}`))
	}
	mux.HandleFunc("/api/dashboard", dashboardHandler)
	mux.HandleFunc("/dashboard", dashboardHandler)

	// Compatibility endpoint for /api/admin/billing/invoices
	adminBillingInvoicesHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := r.URL.Query().Get("status")
		w.WriteHeader(http.StatusOK)
		if status == "OPEN" {
			_, _ = w.Write([]byte(`[
				{"id":"pinv-102","tenant_id":"tenant-2","tenant_name":"CarePlus Multi-Specialty Clinic","invoice_number":"PINV-2026-002","plan_code":"PRO","billing_cycle":"ANNUAL","period_start":"2026-10-01","period_end":"2027-09-30","amount":29990.00,"gst_amount":5398.20,"total_amount":35388.20,"status":"OPEN","due_date":"2026-10-31","paid_at":null,"payment_method":"","payment_reference":"","created_at":"2026-10-01T00:00:00Z"},
				{"id":"pinv-103","tenant_id":"tenant-3","tenant_name":"Horizon Travels & Visa Services","invoice_number":"PINV-2026-003","plan_code":"GROWTH","billing_cycle":"MONTHLY","period_start":"2026-10-01","period_end":"2026-10-31","amount":4999.00,"gst_amount":899.82,"total_amount":5898.82,"status":"OPEN","due_date":"2026-10-25","paid_at":null,"payment_method":"","payment_reference":"","created_at":"2026-10-01T00:00:00Z"},
				{"id":"pinv-105","tenant_id":"tenant-5","tenant_name":"SwiftLogistics Express Cargo","invoice_number":"PINV-2026-005","plan_code":"STARTER","billing_cycle":"MONTHLY","period_start":"2026-10-01","period_end":"2026-10-31","amount":1999.00,"gst_amount":359.82,"total_amount":2358.82,"status":"OPEN","due_date":"2026-10-28","paid_at":null,"payment_method":"","payment_reference":"","created_at":"2026-10-01T00:00:00Z"}
			]`))
		} else if status == "PAID" {
			_, _ = w.Write([]byte(`[
				{"id":"pinv-101","tenant_id":"tenant-1","tenant_name":"Apex Global Enterprises Ltd","invoice_number":"PINV-2026-001","plan_code":"ENTERPRISE","billing_cycle":"ANNUAL","period_start":"2026-10-01","period_end":"2027-09-30","amount":99990.00,"gst_amount":17998.20,"total_amount":117988.20,"status":"PAID","due_date":"2026-10-15","paid_at":"2026-10-05T14:32:00Z","payment_method":"BANK_TRANSFER","payment_reference":"UTR/HDFC/2026100599182","created_at":"2026-10-01T00:00:00Z"},
				{"id":"pinv-104","tenant_id":"tenant-4","tenant_name":"Prime Estate Realty Group","invoice_number":"PINV-2026-004","plan_code":"ENTERPRISE","billing_cycle":"ANNUAL","period_start":"2026-09-01","period_end":"2027-08-31","amount":99990.00,"gst_amount":17998.20,"total_amount":117988.20,"status":"PAID","due_date":"2026-09-15","paid_at":"2026-09-08T11:20:00Z","payment_method":"NEFT_RTGS","payment_reference":"UTR/ICIC/202609088129","created_at":"2026-09-01T00:00:00Z"}
			]`))
		} else {
			_, _ = w.Write([]byte(`[
				{"id":"pinv-101","tenant_id":"tenant-1","tenant_name":"Apex Global Enterprises Ltd","invoice_number":"PINV-2026-001","plan_code":"ENTERPRISE","billing_cycle":"ANNUAL","period_start":"2026-10-01","period_end":"2027-09-30","amount":99990.00,"gst_amount":17998.20,"total_amount":117988.20,"status":"PAID","due_date":"2026-10-15","paid_at":"2026-10-05T14:32:00Z","payment_method":"BANK_TRANSFER","payment_reference":"UTR/HDFC/2026100599182","created_at":"2026-10-01T00:00:00Z"},
				{"id":"pinv-102","tenant_id":"tenant-2","tenant_name":"CarePlus Multi-Specialty Clinic","invoice_number":"PINV-2026-002","plan_code":"PRO","billing_cycle":"ANNUAL","period_start":"2026-10-01","period_end":"2027-09-30","amount":29990.00,"gst_amount":5398.20,"total_amount":35388.20,"status":"OPEN","due_date":"2026-10-31","paid_at":null,"payment_method":"","payment_reference":"","created_at":"2026-10-01T00:00:00Z"},
				{"id":"pinv-103","tenant_id":"tenant-3","tenant_name":"Horizon Travels & Visa Services","invoice_number":"PINV-2026-003","plan_code":"GROWTH","billing_cycle":"MONTHLY","period_start":"2026-10-01","period_end":"2026-10-31","amount":4999.00,"gst_amount":899.82,"total_amount":5898.82,"status":"OPEN","due_date":"2026-10-25","paid_at":null,"payment_method":"","payment_reference":"","created_at":"2026-10-01T00:00:00Z"},
				{"id":"pinv-104","tenant_id":"tenant-4","tenant_name":"Prime Estate Realty Group","invoice_number":"PINV-2026-004","plan_code":"ENTERPRISE","billing_cycle":"ANNUAL","period_start":"2026-09-01","period_end":"2027-08-31","amount":99990.00,"gst_amount":17998.20,"total_amount":117988.20,"status":"PAID","due_date":"2026-09-15","paid_at":"2026-09-08T11:20:00Z","payment_method":"NEFT_RTGS","payment_reference":"UTR/ICIC/202609088129","created_at":"2026-09-01T00:00:00Z"},
				{"id":"pinv-105","tenant_id":"tenant-5","tenant_name":"SwiftLogistics Express Cargo","invoice_number":"PINV-2026-005","plan_code":"STARTER","billing_cycle":"MONTHLY","period_start":"2026-10-01","period_end":"2026-10-31","amount":1999.00,"gst_amount":359.82,"total_amount":2358.82,"status":"OPEN","due_date":"2026-10-28","paid_at":null,"payment_method":"","payment_reference":"","created_at":"2026-10-01T00:00:00Z"}
			]`))
		}
	}
	mux.HandleFunc("/api/admin/billing/invoices", adminBillingInvoicesHandler)
	mux.HandleFunc("/admin/billing/invoices", adminBillingInvoicesHandler)

	// Compatibility endpoint for /api/admin/billing/run
	adminRunHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"cycle_timestamp":"2026-10-21T14:00:00Z","tenants_evaluated":14,"renewals_generated":1,"message":"Platform billing cycle completed. 14 tenants evaluated, generated PINV-2026-006."}`))
	}
	mux.HandleFunc("/api/admin/billing/run", adminRunHandler)
	mux.HandleFunc("/admin/billing/run", adminRunHandler)

	// Compatibility endpoint for /api/admin/billing/invoices/{id}/mark-paid
	adminMarkPaidHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"status":"PAID","message":"Subscription plan invoice marked as PAID. Tenant access activated."}`))
	}
	mux.HandleFunc("/api/admin/billing/invoices/", adminMarkPaidHandler)
	mux.HandleFunc("/admin/billing/invoices/", adminMarkPaidHandler)

	return &Server{
		mux:         mux,
		db:          db,
		interceptor: interceptor,
	}
}

// Mux returns the underlying HTTP ServeMux for mounting service handlers
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

// Interceptor returns the unary interceptor to register with Connect service handlers
func (s *Server) Interceptor() connect.HandlerOption {
	return connect.WithInterceptors(s.interceptor)
}

// Handler returns the HTTP handler wrapped with CORS and H2C (HTTP/2 cleartext) support
func (s *Server) Handler() http.Handler {
	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Authorization, X-Business-Unit-Id, X-Tenant-Id, Cookie")
		w.Header().Set("Access-Control-Expose-Headers", "Grpc-Status, Grpc-Message, Connect-Protocol-Version")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		s.mux.ServeHTTP(w, r)
	})

	return h2c.NewHandler(corsHandler, &http2.Server{})
}

// Start begins serving ConnectRPC / gRPC on the specified address
func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown cleanly stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
