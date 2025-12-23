package handlers

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// DocsHandler handles API documentation endpoints
type DocsHandler struct {
	scalarHTML    []byte
	scalarETag    string
	oas3Path      string
	docsGenerated bool
}

// NewDocsHandler creates a new documentation handler
func NewDocsHandler() *DocsHandler {
	scalarPath := filepath.Join("docs", "scalar.html")
	scalarHTML, err := os.ReadFile(scalarPath)
	if err != nil {
		// If file doesn't exist during initialization, create empty handler
		// The file will be available after build
		scalarHTML = []byte{}
	}

	etag := generateETag(scalarHTML)

	return &DocsHandler{
		scalarHTML:    scalarHTML,
		scalarETag:    etag,
		oas3Path:      filepath.Join("docs", "swagger.json"),
		docsGenerated: fileExists(filepath.Join("docs", "swagger.json")),
	}
}

// ServeScalarUI serves the Scalar HTML page
// @Summary API Documentation UI
// @Description Serves the interactive Scalar documentation interface
// @Tags Documentation
// @Produce html
// @Success 200 {string} string "HTML page"
// @Router /docs [get]
func (h *DocsHandler) ServeScalarUI(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("Expires", "0")

	if h.scalarETag != "" {
		c.Response().Header().Set("ETag", h.scalarETag)
		if match := c.Request().Header.Get("If-None-Match"); match != "" && match == h.scalarETag {
			return c.NoContent(http.StatusNotModified)
		}
	}

	return c.HTMLBlob(http.StatusOK, h.scalarHTML)
}

// ServeOAS3JSON serves the OpenAPI specification file
// This endpoint is called by Scalar to load the API specification
func (h *DocsHandler) ServeOAS3JSON(c echo.Context) error {
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Response().Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.Response().Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
	return c.File(h.oas3Path)
}

// generateETag creates an ETag hash for cache control
func generateETag(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	hash := md5.Sum(data)
	return fmt.Sprintf("\"%x\"", hash)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
