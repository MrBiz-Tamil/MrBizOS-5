package grpcserver

import (
	"context"
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
