package handler

import (
	"net/http"
	"strconv"
	"time"

	"supplier-service/internal/model"
	"supplier-service/pkg/database"
	"supplier-service/pkg/logger"
	"supplier-service/prometheus"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// SupplierRequest defines the structure for supplier creation/update requests
type SupplierRequest struct {
	Name          string `json:"name" validate:"required"`
	Code          string `json:"code" validate:"required"`
	ContactPerson string `json:"contact_person"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Address       string `json:"address"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	PostalCode    string `json:"postal_code"`
	TaxID         string `json:"tax_id"`
	PaymentTerms  string `json:"payment_terms"`
	Notes         string `json:"notes"`
	IsActive      bool   `json:"is_active"`
	Rating        int    `json:"rating"`
	TenantID      uint   `json:"tenant_id" validate:"required"`
}

// CreateSupplier creates a new supplier for the current tenant
func CreateSupplier(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Creating new supplier")
	prometheus.RecordSupplierOperation("create")

	var req SupplierRequest
	if err := c.Bind(&req); err != nil {
		log.Error("Invalid request data", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request data",
		})
	}

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error": "authentication required",
		})
	}

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		prometheus.TenantContextMissingCounter.Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	// Override the tenant ID in the request with the one from the JWT token
	// This ensures users can't create suppliers for other tenants
	req.TenantID = tenantID

	log.Info("Supplier creation request",
		zap.String("name", req.Name),
		zap.String("code", req.Code),
		zap.Uint("tenant_id", req.TenantID))

	// Check if supplier with same code exists in the same tenant
	var count int64
	database.GetDB().Model(&model.Supplier{}).
		Where("code = ? AND tenant_id = ?", req.Code, req.TenantID).
		Count(&count)
	if count > 0 {
		log.Warn("Supplier with this code already exists for this tenant",
			zap.String("code", req.Code),
			zap.Uint("tenant_id", req.TenantID))
		return c.JSON(http.StatusConflict, echo.Map{
			"error": "Supplier with this code already exists for this tenant",
		})
	}

	// Create the supplier
	supplier := model.Supplier{
		Name:          req.Name,
		Code:          req.Code,
		TenantID:      req.TenantID,
		ContactPerson: req.ContactPerson,
		Email:         req.Email,
		Phone:         req.Phone,
		Address:       req.Address,
		City:          req.City,
		State:         req.State,
		Country:       req.Country,
		PostalCode:    req.PostalCode,
		TaxID:         req.TaxID,
		PaymentTerms:  req.PaymentTerms,
		Notes:         req.Notes,
		IsActive:      req.IsActive,
		Rating:        req.Rating,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("insert")(time.Now())

	result := database.GetDB().Create(&supplier)
	if result.Error != nil {
		log.Error("Failed to create supplier",
			zap.String("name", req.Name),
			zap.String("code", req.Code),
			zap.Uint("tenant_id", req.TenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to create supplier",
		})
	}

	// Update supplier count metric
	go updateSupplierCount(tenantID)

	log.Info("Supplier created successfully",
		zap.Uint("id", supplier.ID),
		zap.String("name", supplier.Name),
		zap.String("code", supplier.Code),
		zap.Uint("tenant_id", supplier.TenantID))
	return c.JSON(http.StatusCreated, supplier)
}

// GetSupplier retrieves a supplier by ID for the current tenant
func GetSupplier(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RecordSupplierOperation("get")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Error("Invalid supplier ID", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid supplier ID",
		})
	}

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		prometheus.TenantContextMissingCounter.Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	log.Info("Getting supplier by ID",
		zap.Uint64("supplier_id", id),
		zap.Uint("tenant_id", tenantID))

	// Track DB operations
	defer prometheus.TrackDBOperation("query")(time.Now())

	var supplier model.Supplier
	result := database.GetDB().Where("id = ? AND tenant_id = ?", id, tenantID).First(&supplier)
	if result.Error != nil {
		log.Error("Supplier not found or does not belong to tenant",
			zap.Uint64("supplier_id", id),
			zap.Uint("tenant_id", tenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Supplier not found",
		})
	}

	log.Info("Supplier retrieved successfully",
		zap.Uint64("supplier_id", id),
		zap.String("supplier_name", supplier.Name),
		zap.String("supplier_code", supplier.Code),
		zap.Uint("tenant_id", supplier.TenantID))
	return c.JSON(http.StatusOK, supplier)
}

// ListSuppliers retrieves all suppliers for the current tenant
func ListSuppliers(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Listing suppliers with filters")
	prometheus.RecordSupplierOperation("list")

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		prometheus.TenantContextMissingCounter.Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	// Parse query parameters for pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20 // Default limit
	}

	offset := (page - 1) * limit

	// Handle query parameters for filtering
	db := database.GetDB()
	query := db.Where("tenant_id = ?", tenantID)
	log.Info("Filtering suppliers by tenant", zap.Uint("tenant_id", tenantID))

	// Filter by active status if specified
	isActive := c.QueryParam("is_active")
	if isActive != "" {
		active, err := strconv.ParseBool(isActive)
		if err == nil {
			query = query.Where("is_active = ?", active)
			log.Info("Filtering suppliers by active status", zap.Bool("is_active", active))
		} else {
			log.Warn("Invalid is_active parameter", zap.String("value", isActive), zap.Error(err))
		}
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Retrieve suppliers from database with pagination and filters
	var suppliers []model.Supplier
	result := query.
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&suppliers)

	if result.Error != nil {
		log.Error("Failed to retrieve suppliers",
			zap.Uint("tenant_id", tenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to retrieve suppliers",
		})
	}

	// Count total suppliers for pagination info
	var total int64
	query.Model(&model.Supplier{}).Count(&total)

	log.Info("Suppliers retrieved successfully",
		zap.Int("count", len(suppliers)),
		zap.Int64("total", total),
		zap.Uint("tenant_id", tenantID))

	return c.JSON(http.StatusOK, echo.Map{
		"suppliers": suppliers,
		"pagination": echo.Map{
			"current_page": page,
			"limit":        limit,
			"total":        total,
			"total_pages":  (int(total) + limit - 1) / limit,
		},
	})
}

// UpdateSupplier updates an existing supplier for the current tenant
func UpdateSupplier(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RecordSupplierOperation("update")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Error("Invalid supplier ID", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid supplier ID",
		})
	}

	log.Info("Updating supplier", zap.Uint64("supplier_id", id))

	var req SupplierRequest
	if err := c.Bind(&req); err != nil {
		log.Error("Invalid request data",
			zap.Uint64("supplier_id", id),
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request data",
		})
	}

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error": "authentication required",
		})
	}

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		prometheus.TenantContextMissingCounter.Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	// Override the tenant ID in the request with the one from the JWT token
	// This ensures users can't update suppliers for other tenants
	req.TenantID = tenantID

	// Find existing supplier and validate tenant ownership
	var supplier model.Supplier
	result := database.GetDB().Where("id = ?", id).First(&supplier)
	if result.Error != nil {
		log.Error("Supplier not found for update",
			zap.Uint64("supplier_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Supplier not found",
		})
	}

	// Ensure supplier belongs to the tenant in JWT token
	if supplier.TenantID != tenantID {
		log.Warn("Unauthorized attempt to update supplier from different tenant",
			zap.Uint64("supplier_id", id),
			zap.Uint("supplier_tenant", supplier.TenantID),
			zap.Uint("request_tenant", tenantID))
		return c.JSON(http.StatusForbidden, echo.Map{
			"error": "You don't have permission to update this supplier",
		})
	}

	oldCode := supplier.Code

	// Check if code is changed and if new code already exists within the same tenant
	if req.Code != supplier.Code {
		log.Info("Supplier code change requested",
			zap.Uint64("supplier_id", id),
			zap.String("old_code", oldCode),
			zap.String("new_code", req.Code))

		var count int64
		database.GetDB().Model(&model.Supplier{}).
			Where("code = ? AND id != ? AND tenant_id = ?", req.Code, id, tenantID).
			Count(&count)
		if count > 0 {
			log.Warn("Supplier with this code already exists for this tenant",
				zap.String("code", req.Code),
				zap.Uint("tenant_id", tenantID))
			return c.JSON(http.StatusConflict, echo.Map{
				"error": "Supplier with this code already exists for this tenant",
			})
		}
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("update")(time.Now())

	// Update supplier fields
	supplier.Name = req.Name
	supplier.Code = req.Code
	supplier.ContactPerson = req.ContactPerson
	supplier.Email = req.Email
	supplier.Phone = req.Phone
	supplier.Address = req.Address
	supplier.City = req.City
	supplier.State = req.State
	supplier.Country = req.Country
	supplier.PostalCode = req.PostalCode
	supplier.TaxID = req.TaxID
	supplier.PaymentTerms = req.PaymentTerms
	supplier.Notes = req.Notes
	supplier.IsActive = req.IsActive
	supplier.Rating = req.Rating
	supplier.UpdatedBy = userID
	// TenantID remains unchanged - can't change tenant ownership

	// Save changes
	result = database.GetDB().Save(&supplier)
	if result.Error != nil {
		log.Error("Failed to update supplier",
			zap.Uint64("supplier_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to update supplier",
		})
	}

	log.Info("Supplier updated successfully",
		zap.Uint64("supplier_id", id),
		zap.String("name", supplier.Name),
		zap.String("old_code", oldCode),
		zap.String("new_code", supplier.Code),
		zap.Uint("tenant_id", supplier.TenantID))
	return c.JSON(http.StatusOK, supplier)
}

// DeleteSupplier handles deleting a supplier (soft delete)
func DeleteSupplier(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RecordSupplierOperation("delete")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Error("Invalid supplier ID", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid supplier ID",
		})
	}

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		prometheus.TenantContextMissingCounter.Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	log.Info("Deleting supplier",
		zap.Uint64("supplier_id", id),
		zap.Uint("tenant_id", tenantID))

	// Get supplier details before deleting and verify tenant ownership
	var supplier model.Supplier
	preResult := database.GetDB().Where("id = ? AND tenant_id = ?", id, tenantID).First(&supplier)
	if preResult.Error != nil {
		log.Warn("Supplier not found or does not belong to tenant",
			zap.Uint64("supplier_id", id),
			zap.Uint("tenant_id", tenantID),
			zap.Error(preResult.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Supplier not found",
		})
	}

	log.Info("Found supplier to delete",
		zap.Uint64("supplier_id", id),
		zap.String("name", supplier.Name),
		zap.String("code", supplier.Code),
		zap.Uint("tenant_id", supplier.TenantID))

	// Track DB operations
	defer prometheus.TrackDBOperation("delete")(time.Now())

	// Perform soft delete
	result := database.GetDB().Delete(&supplier)
	if result.Error != nil {
		log.Error("Failed to delete supplier",
			zap.Uint64("supplier_id", id),
			zap.Uint("tenant_id", supplier.TenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to delete supplier",
		})
	}

	// Update supplier count metric
	go updateSupplierCount(tenantID)

	log.Info("Supplier deleted successfully",
		zap.Uint64("supplier_id", id),
		zap.Uint("tenant_id", supplier.TenantID),
		zap.Int64("rows_affected", result.RowsAffected))
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Supplier deleted successfully",
	})
}

// Helper function to update supplier count metrics
func updateSupplierCount(tenantID uint) {
	// Get tenant name
	var tenantName string
	tenantResult := database.GetDB().Table("tenants").
		Select("name").
		Where("id = ?", tenantID).
		Row()
	tenantResult.Scan(&tenantName)

	// Count active suppliers for the tenant
	var count int64
	database.GetDB().Model(&model.Supplier{}).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Count(&count)

	// Update the metric
	prometheus.UpdateSuppliersPerTenant(tenantID, tenantName, int(count))

	// Count distinct tenants with active suppliers
	var activeTenants int64
	database.GetDB().Model(&model.Supplier{}).
		Distinct("tenant_id").
		Where("is_active = ?", true).
		Count(&activeTenants)

	// Update active tenants metric
	prometheus.UpdateActiveTenants(int(activeTenants))
}
