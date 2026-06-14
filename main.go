package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	DbDSN = "postgres://username:password@localhost:5432/auth_db?sslmode=disable"
	Port  = ":8081"
)

var jwtKey = []byte("chiave-super-segreta")

type Comune struct {
	Nome     string  `json:"nome"`
	Abitanti float64 `json:"abitanti"`
}

type Notizia struct {
	Giornale  string `json:"giornale"`
	Titolo    string `json:"titolo"`
	Comune    string `json:"comune"`
	Contenuto string `json:"contenuto"`
	Tipologia string `json:"tipologia"`
	Data      string `json:"data"`
}

type Env struct {
	db *sql.DB
}

type CustomClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func main() {
	db, err := sql.Open("pgx", DbDSN)

	if err != nil {
		log.Fatalf("Errore configurazione DB: %v", err)
	}
	defer db.Close()

	env := &Env{db: db}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /notizie", env.addNews)

	fmt.Printf("Identity Service (HTTP) pronto sulla porta %s\n", Port)
	log.Fatal(http.ListenAndServe(Port, mux))
}

func (env *Env) addNews(w http.ResponseWriter, r *http.Request) {
	//controllo dei permessi
	auth := r.Header.Get("Authorization")

	if auth == "" {
		http.Error(w, "Token Mancante", http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "Token invalido", http.StatusUnauthorized)
		return
	}

	tokenStr := auth[7:]
	claims := &CustomClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Token Non valido", http.StatusUnauthorized)
		return
	}

	if claims.Role != "admin" {
		http.Error(w, "Permessi insufficienti", http.StatusForbidden)
		return
	}

	var req Notizia
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON non valido", http.StatusBadRequest)
		return
	}

	var comuneEsiste bool
	checkQuery := `SELECT EXISTS(SELECT nome FROM comuni WHERE nome = $1)`

	err = env.db.QueryRow(checkQuery, req.Comune).Scan(&comuneEsiste)
	if err != nil {
		http.Error(w, "Errore nella comunicazione con il database", http.StatusInternalServerError)
		return
	}

	if !comuneEsiste {
		http.Error(w, "Errore: Il comune specificato non esiste nel database", http.StatusBadRequest)
		return
	}

	insertQuery := `INSERT INTO notizie (titolo, contenuto, comune_id) VALUES ($1, $2, $3) RETURNING id`

	err = env.db.QueryRow(insertQuery, req.Titolo, req.Contenuto, req.Comune).Scan(&req.Comune)
	if err != nil {
		http.Error(w, "Errore durante il salvataggio della notizia", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)

}
