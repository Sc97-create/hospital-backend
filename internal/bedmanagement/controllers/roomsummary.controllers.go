package controllers

import (
	"hospital-backend/internal/bedmanagement/services"

	"github.com/gofiber/fiber/v2"
)

type IRoomSummaryController interface {
	GetRoomSummaryByRoomType(c *fiber.Ctx) error
}

type RoomSummaryController struct {
	RoomSummaryService *services.RoomSummaryService
}

func NewRoomSummaryController(roomSummaryService *services.RoomSummaryService) *RoomSummaryController {
	return &RoomSummaryController{RoomSummaryService: roomSummaryService}
}
