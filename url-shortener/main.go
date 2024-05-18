package main

import (
	"url-shortener/log"
	"url-shortener/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

const logsDir = "./logs"

func setupRoutes(app *fiber.App) {
	app.Get("/:url", routes.Resolve)
	app.Post("/api/v1", routes.Shorten)
}

func main() {
	log.InitLogger()
	log.Info().Msgf("Initialized Logger")

	err := godotenv.Load()
	if err != nil {
		log.Fatal().Msgf("Could not load environment file")
	}

	app := fiber.New()
	app.Use(logger.New())
	setupRoutes(app)
	app.Listen(":3000")
}
