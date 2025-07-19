package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mheers/godtemplate/invoicerenderer"
	"github.com/sirupsen/logrus"
)

// InvoiceRequest represents the incoming request structure
type InvoiceRequest struct {
	Invoice invoicerenderer.Invoice       `json:"invoice"`
	Items   []invoicerenderer.InvoiceItem `json:"items"`
}

// Server holds the configuration for the HTTP server
type Server struct {
	Port         string
	TemplatePath string
	Logger       *logrus.Logger
}

// NewServer creates a new server instance
func NewServer(port, templatePath string) *Server {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &Server{
		Port:         port,
		TemplatePath: templatePath,
		Logger:       logger,
	}
}

// Start starts the web service that handles POST requests and answers with rendered invoice as PDF
func (s *Server) Start() error {
	http.HandleFunc("/render", s.renderInvoiceHandler)
	http.HandleFunc("/render-base64", s.renderBase64Handler)
	http.HandleFunc("/health", s.healthHandler)

	s.Logger.Infof("Starting server on port %s", s.Port)
	s.Logger.Infof("Template path: %s", s.TemplatePath)
	s.Logger.Info("Available endpoints:")
	s.Logger.Info("  POST /render - Render invoice from JSON data")
	s.Logger.Info("  POST /render-base64 - Render invoice from base64 encoded JSON")
	s.Logger.Info("  GET /health - Health check")

	return http.ListenAndServe(":"+s.Port, nil)
}

// renderInvoiceHandler handles POST requests to render invoices as PDF
func (s *Server) renderInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.Logger.Errorf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON request
	var req InvoiceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.Logger.Errorf("Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate template path
	if _, err := os.Stat(s.TemplatePath); os.IsNotExist(err) {
		s.Logger.Errorf("Template file not found: %s", s.TemplatePath)
		http.Error(w, "Template file not found", http.StatusInternalServerError)
		return
	}

	// Create temporary files for processing
	tempDir := os.TempDir()
	timestamp := time.Now().UnixNano()
	tempODT := filepath.Join(tempDir, fmt.Sprintf("invoice_%d.odt", timestamp))
	tempPDF := filepath.Join(tempDir, fmt.Sprintf("invoice_%d.pdf", timestamp))

	// Clean up temporary files
	defer func() {
		os.Remove(tempODT)
		os.Remove(tempPDF)
	}()

	// Render invoice to ODT
	if err := invoicerenderer.RenderInvoice(s.TemplatePath, req.Invoice, req.Items, tempODT); err != nil {
		s.Logger.Errorf("Failed to render invoice: %v", err)
		http.Error(w, fmt.Sprintf("Failed to render invoice: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert ODT to PDF
	if err := invoicerenderer.ConvertODTToPDF(tempODT, tempPDF); err != nil {
		s.Logger.Errorf("Failed to convert ODT to PDF: %v", err)
		http.Error(w, fmt.Sprintf("Failed to convert to PDF: %v", err), http.StatusInternalServerError)
		return
	}

	// Read the generated PDF
	pdfContent, err := os.ReadFile(tempPDF)
	if err != nil {
		s.Logger.Errorf("Failed to read generated PDF: %v", err)
		http.Error(w, "Failed to read generated PDF", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=invoice_%s.pdf", req.Invoice.DocumentNumber))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfContent)))

	// Write PDF content to response
	if _, err := w.Write(pdfContent); err != nil {
		s.Logger.Errorf("Failed to write PDF to response: %v", err)
		return
	}

	s.Logger.Infof("Successfully rendered invoice %s", req.Invoice.DocumentNumber)
}

// renderBase64Handler handles POST requests with base64 encoded JSON data
func (s *Server) renderBase64Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body (should contain base64 encoded JSON)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.Logger.Errorf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Decode base64 JSON
	var req InvoiceRequest
	if err := invoicerenderer.DecodeBase64JSON(string(bytes.TrimSpace(body)), &req); err != nil {
		s.Logger.Errorf("Failed to decode base64 JSON: %v", err)
		http.Error(w, "Invalid base64 JSON format", http.StatusBadRequest)
		return
	}

	// Create a new request with decoded data and forward to main handler
	jsonData, err := json.Marshal(req)
	if err != nil {
		s.Logger.Errorf("Failed to marshal decoded data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create new request with decoded JSON
	newReq, err := http.NewRequest("POST", "/render", bytes.NewReader(jsonData))
	if err != nil {
		s.Logger.Errorf("Failed to create new request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	newReq.Header.Set("Content-Type", "application/json")

	// Forward to main handler
	s.renderInvoiceHandler(w, newReq)
}

// healthHandler provides a simple health check endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"template":  s.TemplatePath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
