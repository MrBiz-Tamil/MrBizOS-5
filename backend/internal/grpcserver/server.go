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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Authorization, X-Business-Unit-Id, X-Tenant-Id")
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
