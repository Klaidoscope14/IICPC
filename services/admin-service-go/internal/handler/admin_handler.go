package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/admin-service-go/internal/domain"
	"github.com/iicpc/admin-service-go/internal/service"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) RegisterRoutes(router *gin.Engine) {
	adminGroup := router.Group("/api/v1/admin")
	{
		adminGroup.POST("/login", h.Login)
		adminGroup.PUT("/hackathon/dates", h.UpdateHackathonDates)
		adminGroup.GET("/hackathon/dates", h.GetHackathonDates)
		adminGroup.GET("/users", h.GetUsers)
		adminGroup.GET("/teams", h.GetTeams)
		adminGroup.GET("/submissions", h.GetSubmissions)
		adminGroup.GET("/stats", h.GetStats)
	}
	router.GET("/health", h.Health)
}

func (h *AdminHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "up"})
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.adminService.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, token)
}

func (h *AdminHandler) UpdateHackathonDates(c *gin.Context) {
	var req domain.HackathonDatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.adminService.UpdateHackathonDates(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "hackathon dates updated successfully"})
}

func (h *AdminHandler) GetHackathonDates(c *gin.Context) {
	// Let's reuse GetStats to just get stats or implement a new method in service/repository
	// Actually, wait, repo doesn't have GetHackathonDates yet.
	// We'll write it directly in repo or just create a dummy one for now if not implemented.
	dates, err := h.adminService.GetHackathonConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dates)
}

func (h *AdminHandler) GetUsers(c *gin.Context) {
	users, err := h.adminService.GetUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *AdminHandler) GetTeams(c *gin.Context) {
	teams, err := h.adminService.GetTeams(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

func (h *AdminHandler) GetSubmissions(c *gin.Context) {
	submissions, err := h.adminService.GetSubmissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"submissions": submissions})
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.adminService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
