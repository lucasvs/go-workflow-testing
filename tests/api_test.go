package main_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
)

type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

func setupTestDB(t *testing.T) *sql.DB {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Fatal("DB_URL environment variable not set")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to the database: %v", err)
	}

	// Create the books table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS books (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			author TEXT NOT NULL,
			year INT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create books table: %v", err)
	}

	// Truncate the books table to ensure it's clean
	_, err = db.Exec("TRUNCATE TABLE books RESTART IDENTITY")
	if err != nil {
		t.Fatalf("Failed to truncate books table: %v", err)
	}

	return db
}

func TestCreateBookViaAPI(t *testing.T) {
	// Setup the test database
	db := setupTestDB(t)
	defer db.Close()

	// URL da API que está rodando no container
	apiURL := "http://localhost:8080" // Substitua pelo endereço correto se necessário

	// Dados do livro a ser criado
	newBook := `{"title": "1984", "author": "George Orwell", "year": 1949}`

	// Faz a requisição POST para o endpoint /books
	resp, err := http.Post(fmt.Sprintf("%s/books", apiURL), "application/json", strings.NewReader(newBook))
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Verifica se a resposta foi bem-sucedida
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Decodifica a resposta para obter o livro criado
	var createdBook Book
	err = json.NewDecoder(resp.Body).Decode(&createdBook)
	assert.NoError(t, err)

	// Verifica se o livro foi salvo corretamente no banco de dados
	var dbBook Book
	err = db.QueryRow("SELECT id, title, author, year FROM books WHERE id = $1", createdBook.ID).Scan(&dbBook.ID, &dbBook.Title, &dbBook.Author, &dbBook.Year)
	assert.NoError(t, err)

	// Compara os dados do livro criado com os dados no banco de dados
	assert.Equal(t, createdBook.ID, dbBook.ID)
	assert.Equal(t, "1984", dbBook.Title)
	assert.Equal(t, "George Orwell", dbBook.Author)
	assert.Equal(t, 1949, dbBook.Year)
}
