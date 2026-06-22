package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"zadia-host/db"
	"zadia-host/models"
)

func GetEnvVars(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
	}

	envVars, err := db.GetEnvVars(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(envVars)
}

func CreateEnvVar(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
	}

	var req models.CreateEnvVarRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Corps de requête invalide"})
	}

	if req.Key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "La clé est requise"})
	}

	envID, err := db.CreateEnvVar(id, req.Key, req.Value)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"id":     envID,
		"vps_id": id,
		"key":    req.Key,
		"value":  req.Value,
	})
}

func DeleteEnvVar(c *fiber.Ctx) error {
	envID, err := strconv.ParseInt(c.Params("envId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID variable invalide"})
	}

	if err := db.DeleteEnvVar(envID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Variable supprimée"})
}
