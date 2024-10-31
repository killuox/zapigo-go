package main

import (
	"log"
	"net/http"

	"github.com/Killuox/zapigo-go/db"
	"github.com/Killuox/zapigo-go/slack"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load the .env file
	godotenv.Load()

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"https://slack.com"},
		AllowMethods: []string{echo.GET, echo.POST},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to zapigo!")
	})

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	con, err := db.Connect()
	if err != nil {
		log.Fatal("Error connecting to the database")
	}
	db.CreateTable()
	defer db.Close(con)

	e.POST("/command/go", slack.GoCommand)
	e.POST("/command/add", slack.AddCommand)
	e.POST("/command/edit", slack.EditCommand)
	e.POST("/command/delete", slack.DeleteCommand)
	e.POST("/command/list", slack.ListCommand)
	e.POST("/interaction", slack.Interaction)
	e.POST("/event", slack.OnEvent)

	// Start the Echo server and listen on port 8080
	port := ":8080"
	if err := e.Start(port); err != nil {
		e.Logger.Fatal("Shutting down the server due to:", err)
	}
}
