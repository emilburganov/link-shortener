package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

func main() {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	router := gin.Default()

	router.GET("/health", healthHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		log.Fatal("APP_PORT is not set")
	}

	err = http.ListenAndServe(":"+port, router)
	if err != nil {
		log.Fatalf("error: %s", err.Error())
	}
}
