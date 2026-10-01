package handler

import (
	"basic-app/dto"
	"basic-app/models"
	"basic-app/services"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MenuHandler struct {
	menuService *services.MenuService
}

func NewMenuHandler(
	menuService *services.MenuService,
) *MenuHandler {
	return &MenuHandler{
		menuService: menuService,
	}
}

func (h *MenuHandler) Create(c *gin.Context) {
	var req dto.CreateMenuRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	menu, err := h.menuService.Create(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidMenuName),
			errors.Is(err, services.ErrInvalidMenuLocation),
			errors.Is(err, services.ErrInvalidMenuItem),
			errors.Is(err, services.ErrMenuDepthExceeded):

			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrMenuAlreadyExists):

			c.JSON(http.StatusConflict, gin.H{
				"error": "menu already exists for this location",
			})

		case errors.Is(err, services.ErrPageNotFound):

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "referenced page not found",
			})

		default:

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to create menu",
			})
		}

		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "menu created successfully",
		"menu":    menu,
	})
}

func (h *MenuHandler) Get(c *gin.Context) {
	id := c.Param("id")

	menu, err := h.menuService.Get(
		c.Request.Context(),
		id,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMenuNotFound):

			c.JSON(http.StatusNotFound, gin.H{
				"error": "menu not found",
			})

		default:

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to get menu",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"menu": menu,
	})
}

func (h *MenuHandler) List(c *gin.Context) {
	menus, err := h.menuService.List(
		c.Request.Context(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to get menus",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"menus": menus,
	})
}

func (h *MenuHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateMenuRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	menu, err := h.menuService.Update(
		c.Request.Context(),
		id,
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMenuNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "menu not found",
			})

		case errors.Is(err, services.ErrInvalidMenuName):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to update menu",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "menu updated successfully",
		"data":    menu,
	})
}

func (h *MenuHandler) GetPublicByLocation(c *gin.Context) {
	location := models.MenuLocation(
		c.Param("location"),
	)

	menu, err := h.menuService.GetPublicByLocation(
		c.Request.Context(),
		location,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidMenuLocation):

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid menu location",
			})

		case errors.Is(err, services.ErrMenuNotFound):

			c.JSON(http.StatusNotFound, gin.H{
				"error": "menu not found",
			})

		case errors.Is(err, services.ErrMenuInactive):

			c.JSON(http.StatusNotFound, gin.H{
				"error": "menu is inactive",
			})

		default:

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to get menu",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"menu": menu,
	})
}

func (h *MenuHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.menuService.Delete(
		c.Request.Context(),
		id,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMenuNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "menu not found",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to delete menu",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "menu deleted successfully",
	})
}

func (h *MenuHandler) AddItem(c *gin.Context) {
	menuID := c.Param("id")

	var req dto.AddMenuItemRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid request",
		})
		return
	}

	menu, err := h.menuService.AddItem(
		c.Request.Context(),
		menuID,
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMenuNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"message": "menu not found",
			})
		case errors.Is(err, services.ErrPageNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"message": "page not found",
			})
		case errors.Is(err, services.ErrInvalidMenuItem):
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid menu item",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to add menu item",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "menu item added successfully",
		"data":    menu,
	})
}

func (h *MenuHandler) UpdateItem(c *gin.Context) {
	menuID := c.Param("id")
	itemID := c.Param("itemId")

	var req dto.UpdateMenuItemRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	menu, err := h.menuService.UpdateItem(
		c.Request.Context(),
		menuID,
		itemID,
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMenuNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "menu not found",
			})

		case errors.Is(err, services.ErrMenuItemNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "menu item not found",
			})

		case errors.Is(err, services.ErrPageNotFound):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "referenced page not found",
			})

		case errors.Is(err, services.ErrInvalidMenuItem):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to update menu item",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "menu item updated successfully",
		"data":    menu,
	})
}
