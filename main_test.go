package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
)

var testDB *sql.DB

// Configura o banco de testes antes de todos os testes
func setupTestDB() {
	// Obtém a URL do banco de dados da variável de ambiente
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		panic("A variável de ambiente DB_URL não está configurada")
	}

	var err error
	testDB, err = sql.Open("pgx", dbURL)
	if err != nil {
		panic("Erro ao conectar ao banco de testes: " + err.Error())
	}

	// Cria a tabela "books" caso não exista
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS books (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			author VARCHAR(255) NOT NULL,
			year INT NOT NULL
		);
	`)
	if err != nil {
		panic("Erro ao criar a tabela: " + err.Error())
	}

	// Limpa as tabelas antes de começar
	_, err = testDB.Exec("TRUNCATE TABLE books RESTART IDENTITY")
	if err != nil {
		panic("Erro ao limpar a tabela: " + err.Error())
	}
}

// Limpa o banco de dados após os testes
func teardownTestDB() {
	testDB.Close()
}

func TestMain(m *testing.M) {
	// Configura e desmonta o banco de testes
	setupTestDB()
	defer teardownTestDB()

	// Executa os testes
	m.Run()
}

func TestCreateBook(t *testing.T) {
	// Configurar o router Gin com o banco de teste
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/books", func(c *gin.Context) {
		var book Book
		if err := c.ShouldBindJSON(&book); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := "INSERT INTO books (title, author, year) VALUES ($1, $2, $3) RETURNING id"
		row := testDB.QueryRow(query, book.Title, book.Author, book.Year)
		if err := row.Scan(&book.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save book: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, book)
	})

	// Configurar payload da requisição
	payload := map[string]interface{}{
		"title":  "Test Book",
		"author": "Test Author",
		"year":   2023,
	}
	jsonPayload, _ := json.Marshal(payload)

	// Criar a requisição HTTP simulada
	req := httptest.NewRequest("POST", "/books", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	// Criar o ResponseRecorder para capturar a resposta
	w := httptest.NewRecorder()

	// Executar a requisição
	router.ServeHTTP(w, req)

	// Validar a resposta HTTP
	assert.Equal(t, http.StatusCreated, w.Code)

	// Validar o estado do banco de dados
	var id int
	var title, author string
	var year int
	err := testDB.QueryRow("SELECT id, title, author, year FROM books WHERE title = $1", "Test Book").Scan(&id, &title, &author, &year)
	assert.NoError(t, err)
	assert.Equal(t, "Test Book", title)
	assert.Equal(t, "Test Author", author)
	assert.Equal(t, 2023, year)
}
