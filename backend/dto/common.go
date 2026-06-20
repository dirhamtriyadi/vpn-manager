package dto

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Stable, machine-readable result/error codes. Clients should branch on these
// rather than on the human-readable message (which may change) or on the HTTP
// status alone (which is coarser). Each code lines up with the HTTP status that
// carries it; see codeForStatus.
const (
	CodeOK                 = "OK"
	CodeCreated            = "CREATED"
	CodeBadRequest         = "BAD_REQUEST"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodePreconditionFailed = "PRECONDITION_FAILED"
	// CodeValidationError is reserved for field-level form validation (422) and
	// always ships an `errors` map. CodeUnprocessable is the generic 422 for a
	// well-formed request that is semantically rejected without field details.
	CodeValidationError    = "VALIDATION_ERROR"
	CodeUnprocessable      = "UNPROCESSABLE_ENTITY"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeInternalError      = "INTERNAL_ERROR"
)

// APIResponse is the standard success envelope returned by JSON API endpoints.
// Single/list response:
//
//	{"success":true,"code":"OK","message":"...","data":...}
//
// Paginated responses add "meta"; partially-applied operations add "warning".
type APIResponse struct {
	Success bool        `json:"success"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    interface{} `json:"meta,omitempty"`
	// Warning is set when the primary operation succeeded and was persisted, but a
	// secondary best-effort step degraded (e.g. the WireGuard kernel reconcile or
	// a VPN runtime apply failed). The status stays 2xx because the resource change
	// was committed; the warning carries the degradation detail for the client.
	Warning string `json:"warning,omitempty"`
}

// ValidationErrors is field name -> list of messages. This shape maps directly
// to React Hook Form's setError(field, ...).
type ValidationErrors map[string][]string

// ErrorResponse is the standard error envelope returned by JSON API endpoints.
// Generic error (400/401/403/404/409/412/500/503):
//
//	{"success":false,"code":"CONFLICT","message":"..."}
//
// Validation error (422) adds field-level details:
//
//	{"success":false,"code":"VALIDATION_ERROR","message":"Validation failed","errors":{"name":["..."]}}
type ErrorResponse struct {
	Success bool        `json:"success"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

type PaginationMeta struct {
	Page      int    `json:"page"`
	PerPage   int    `json:"per_page"`
	Total     int64  `json:"total"`
	LastPage  int    `json:"last_page"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
	Search    string `json:"search,omitempty"`
}

type ListQuery struct {
	Page      int
	PerPage   int
	Offset    int
	SortBy    string
	SortOrder string
	Search    string
}

func Success(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, APIResponse{Success: true, Code: codeForStatus(status), Message: message, Data: data})
}

func SuccessMeta(c *gin.Context, status int, message string, data interface{}, meta interface{}) {
	c.JSON(status, APIResponse{Success: true, Code: codeForStatus(status), Message: message, Data: data, Meta: meta})
}

func OK(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusOK, message, data)
}

func Paginated(c *gin.Context, message string, data interface{}, meta PaginationMeta) {
	SuccessMeta(c, http.StatusOK, message, data, meta)
}

func Created(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusCreated, message, data)
}

func NoData(c *gin.Context, status int, message string) {
	Success(c, status, message, nil)
}

// OKWarn / CreatedWarn / NoDataWarn report a 2xx success whose secondary apply
// step degraded. The resource change was persisted; warning carries the detail
// (e.g. "saved but not applied to kernel: ..."). Prefer these over folding the
// degradation into the message so clients can detect it programmatically.
func OKWarn(c *gin.Context, message string, data interface{}, warning string) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Code: CodeOK, Message: message, Data: data, Warning: warning})
}

func CreatedWarn(c *gin.Context, message string, data interface{}, warning string) {
	c.JSON(http.StatusCreated, APIResponse{Success: true, Code: CodeCreated, Message: message, Data: data, Warning: warning})
}

func NoDataWarn(c *gin.Context, message string, warning string) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Code: CodeOK, Message: message, Warning: warning})
}

func ParsePagination(c *gin.Context) (page int, perPage int, offset int) {
	q := ParseListQuery(c, nil, "")
	return q.Page, q.PerPage, q.Offset
}

func ParseListQuery(c *gin.Context, allowedSorts map[string]string, defaultSort string) ListQuery {
	page := parsePositiveInt(c.Query("page"), 1)
	perPage := parsePositiveInt(c.Query("per_page"), 10)
	if perPage > 100 {
		perPage = 100
	}

	sortBy := strings.TrimSpace(c.Query("sort_by"))
	if sortBy == "" {
		sortBy = defaultSort
	}
	if allowedSorts != nil {
		if _, ok := allowedSorts[sortBy]; !ok {
			sortBy = defaultSort
		}
	}

	sortOrder := strings.ToLower(strings.TrimSpace(c.Query("sort_order")))
	if sortOrder != "desc" {
		sortOrder = "asc"
	}

	return ListQuery{
		Page:      page,
		PerPage:   perPage,
		Offset:    (page - 1) * perPage,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Search:    strings.TrimSpace(c.Query("search")),
	}
}

func (q ListQuery) OrderClause(allowedSorts map[string]string) string {
	column, ok := allowedSorts[q.SortBy]
	if !ok || column == "" {
		return "id asc"
	}
	return column + " " + q.SortOrder
}

func NewPaginationMeta(page int, perPage int, total int64) PaginationMeta {
	lastPage := 1
	if total > 0 {
		lastPage = int((total + int64(perPage) - 1) / int64(perPage))
	}
	return PaginationMeta{Page: page, PerPage: perPage, Total: total, LastPage: lastPage}
}

func NewListMeta(q ListQuery, total int64) PaginationMeta {
	meta := NewPaginationMeta(q.Page, q.PerPage, total)
	meta.SortBy = q.SortBy
	meta.SortOrder = q.SortOrder
	meta.Search = q.Search
	return meta
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// Error writes a generic error envelope. The machine-readable code is derived
// from the HTTP status so every call site stays consistent without restating it.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{Success: false, Code: codeForStatus(status), Message: message})
}

// ErrorCode writes a generic error with an explicit code, for the rare case the
// code should be more specific than the status implies.
func ErrorCode(c *gin.Context, status int, code string, message string) {
	c.JSON(status, ErrorResponse{Success: false, Code: code, Message: message})
}

// ValidationError writes a 422 with field-level details:
//
//	{"success":false,"code":"VALIDATION_ERROR","message":"Validation failed","errors":{"field":["..."]}}
func ValidationError(c *gin.Context, errors ValidationErrors) {
	if errors == nil {
		errors = ValidationErrors{}
	}
	c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
		Success: false,
		Code:    CodeValidationError,
		Message: "Validation failed",
		Errors:  errors,
	})
}

// codeForStatus maps an HTTP status to its stable machine-readable code.
func codeForStatus(status int) string {
	switch status {
	case http.StatusOK:
		return CodeOK
	case http.StatusCreated:
		return CodeCreated
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusPreconditionFailed:
		return CodePreconditionFailed
	case http.StatusUnprocessableEntity:
		// Generic 422; ValidationError() sets CodeValidationError explicitly when
		// field-level details are attached.
		return CodeUnprocessable
	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable
	default:
		switch {
		case status >= 500:
			return CodeInternalError
		case status >= 400:
			return CodeBadRequest
		default:
			return CodeOK
		}
	}
}
