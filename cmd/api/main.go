package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"status":  "success",
		})
	})

	// TODO: Connect to MySQL here
	// TODO: Register Auth & Team routes here

	fmt.Println("Starting server on port 8080")
	if err := router.Run(":8080"); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
