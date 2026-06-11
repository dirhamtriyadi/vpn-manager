package dto

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard success envelope returned by JSON API endpoints.
// Single/list response:
//   {"success":true,"message":"...","data":...}
// Paginated response:
//   {"success":true,"message":"...","data":[...],"meta":{...}}
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    interface{} `json:"meta,omitempty"`
}

// ErrorItem is used for non-field errors (500/404/conflict/etc.).
type ErrorItem struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ValidationErrors is Laravel-style: field name -> list of messages.
// This shape maps directly to React Hook Form's setError(field, ...).
type ValidationErrors map[string][]string

// ErrorResponse is the standard error envelope returned by JSON API endpoints.
// Generic error:
//   {"success":false,"message":"...","errors":[{"message":"..."}]}
// Validation error:
//   {"success":false,"message":"validation failed","errors":{"name":["..."]}}
type ErrorResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors"`
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
	c.JSON(status, APIResponse{Success: true, Message: message, Data: data})
}

func SuccessMeta(c *gin.Context, status int, message string, data interface{}, meta interface{}) {
	c.JSON(status, APIResponse{Success: true, Message: message, Data: data, Meta: meta})
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

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Message: message,
		Errors:  []ErrorItem{{Message: message}},
	})
}

func ErrorWithDetails(c *gin.Context, status int, message string, errors []ErrorItem) {
	if len(errors) == 0 {
		errors = []ErrorItem{{Message: message}}
	}
	c.JSON(status, ErrorResponse{Success: false, Message: message, Errors: errors})
}

func ValidationError(c *gin.Context, errors ValidationErrors) {
	if errors == nil {
		errors = ValidationErrors{}
	}
	c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
		Success: false,
		Message: "validation failed",
		Errors:  errors,
	})
}
