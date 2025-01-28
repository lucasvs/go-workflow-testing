package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

func main() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		panic("DB_URL environment variable not set")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		panic("Failed to connect to the database: " + err.Error())
	}
	defer db.Close()

	// Testa a conexão ao banco
	err = db.Ping()
	if err != nil {
		panic("Failed to ping database: " + err.Error())
	}
	fmt.Println("Database connection successful")

	r := gin.Default()

	r.POST("/books", func(c *gin.Context) {
		var book Book
		if err := c.ShouldBindJSON(&book); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := "INSERT INTO books (title, author, year) VALUES ($1, $2, $3) RETURNING id"
		row := db.QueryRow(query, book.Title, book.Author, book.Year)
		if err := row.Scan(&book.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save book: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, book)
	})

	r.Run(":8080") // Start the server on port 8080
}
