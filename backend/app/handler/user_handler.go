package handler

import (
	"errors"
	"net/http"
	"strings"

	"basic-app/dto"
	"basic-app/services"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(
	userService *services.UserService,
) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) Me(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userIDString, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	user, err := h.userService.GetMe(
		c.Request.Context(),
		userIDString,
	)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to fetch user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userIDString, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req dto.UpdateUserProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.userService.UpdateProfile(
		c.Request.Context(),
		userIDString,
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})

		case errors.Is(err, services.ErrEmailAlreadyExists),
			errors.Is(err, services.ErrAltEmailAlreadyExists),
			errors.Is(err, services.ErrUsernameAlreadyExists),
			errors.Is(err, services.ErrPhoneAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrAltEmailSameAsEmail),
			errors.Is(err, services.ErrNoFieldsToUpdate):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			// Validation errors from the service also reach here.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "profile updated successfully",
		"user":    user,
	})
}

func (h *UserHandler) CreateCustomer(c *gin.Context) {
	var req dto.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.userService.CreateCustomer(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmailAlreadyExists),
			errors.Is(err, services.ErrAltEmailAlreadyExists),
			errors.Is(err, services.ErrUsernameAlreadyExists),
			errors.Is(err, services.ErrPhoneAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrAltEmailSameAsEmail):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			// Validation errors reach here.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}

		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "customer created successfully",
		"user":    user,
	})
}

func (h *UserHandler) UpdateCustomer(c *gin.Context) {
	customerID := c.Param("id")

	var req dto.UpdateUserProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.userService.UpdateCustomer(
		c.Request.Context(),
		customerID,
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidUserID):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "customer not found",
			})

		case errors.Is(err, services.ErrInvalidUserRole):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "target user is not a customer",
			})

		case errors.Is(err, services.ErrEmailAlreadyExists),
			errors.Is(err, services.ErrAltEmailAlreadyExists),
			errors.Is(err, services.ErrUsernameAlreadyExists),
			errors.Is(err, services.ErrPhoneAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrAltEmailSameAsEmail),
			errors.Is(err, services.ErrNoFieldsToUpdate):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "customer updated successfully",
		"user":    user,
	})
}

func (h *UserHandler) ListCustomers(c *gin.Context) {
	var query dto.ListCustomersQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid query parameters",
		})
		return
	}

	result, err := h.userService.ListCustomers(
		c.Request.Context(),
		query,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPage),
			errors.Is(err, services.ErrInvalidLimit):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to fetch customers",
			})
		}

		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *UserHandler) GetCustomerByID(c *gin.Context) {
	customerID := c.Param("id")

	user, err := h.userService.GetCustomerByID(
		c.Request.Context(),
		customerID,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidUserID):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "customer not found",
			})

		case errors.Is(err, services.ErrInvalidUserRole):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "target user is not a customer",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to fetch customer",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"customer": user,
	})
}

func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	adminID := c.GetString("userID")
	targetUserID := strings.TrimSpace(c.Param("id"))

	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user id is required",
		})
		return
	}

	var req dto.UserStatusRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if req.Status == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "status is required",
		})
		return
	}

	err := h.userService.UserStatus(
		c.Request.Context(),
		adminID,
		targetUserID,
		*req.Status,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})

		case errors.Is(err, services.ErrCannotChangeOwnStatus):
			c.JSON(http.StatusForbidden, gin.H{
				"error": "you cannot change your own account status",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user status updated successfully",
		"status":  *req.Status,
	})
}
func (h *UserHandler) ChangeUserPassword(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user id is required",
		})
		return
	}

	var req dto.ChangeUserPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if err := h.userService.ChangeUserPassword(
		c.Request.Context(),
		userID,
		&req,
	); err != nil {
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})

		case errors.Is(err, services.ErrInvalidUserID):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid user id",
			})

		case errors.Is(err, services.ErrInvalidUserRole):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "password can only be reset for customer users",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user password changed successfully",
	})
}

func (h *UserHandler) UpdateProfilePic(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userIDString, ok := userID.(string)
	if !ok || strings.TrimSpace(userIDString) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user identity",
		})
		return
	}

	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid multipart form",
		})
		return
	}

	files := c.Request.MultipartForm.File["profile_pic"]

	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "profile picture is required",
		})
		return
	}

	if len(files) > 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "only one profile picture can be uploaded at a time",
		})
		return
	}

	fileHeader := files[0]

	if err := h.userService.UpdateProfilePic(
		c.Request.Context(),
		userIDString,
		fileHeader,
	); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidUserID):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrInvalidUserRole):
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "profile picture updated successfully",
	})
}
