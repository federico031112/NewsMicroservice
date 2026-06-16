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
	Giornale           string `json:"giornale"`
	Titolo             string `json:"titolo"`
	Comune             string `json:"comune"`
	Contenuto          string `json:"contenuto"`
	Tipologia          string `json:"tipologia"`
	Data_pubblicazione string `json:"data_pubblicazione"`
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

	mux.HandleFunc("POST /api/notizie", env.addNews)
	mux.HandleFunc("GET /api/notizie/titolo/{titolo}", env.getNewsByTitolo)
	mux.HandleFunc("GET /api/notizie/tipologia/{tipologia}", env.getNewsByTipologia)
	mux.HandleFunc("GET /api/notizie/comune/{comune}", env.getNewsByComune)
	mux.HandleFunc("POST /api/comuni", env.insertComune)

	fmt.Printf("Identity Service (HTTP) pronto sulla porta %s\n", Port)
	log.Fatal(http.ListenAndServe(Port, mux))
}

func (env *Env) insertComune(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")

	if auth == "" {
		http.Error(w, "Token mancante", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "Token non valido", http.StatusBadRequest)
		return
	}

	tokenstr := auth[7:]
	claims := &CustomClaims{}

	token, err := jwt.ParseWithClaims(tokenstr, claims, func(token *jwt.Token) (interface{}, error) { return jwtKey, nil })

	if err != nil || !token.Valid {
		http.Error(w, "Token non valido", http.StatusBadRequest)
		return
	}

	if claims.Role != "admin" {
		http.Error(w, "Permessi insufficienti", http.StatusForbidden)
		return
	}

	var comune Comune
	if err := json.NewDecoder(r.Body).Decode(&comune); err != nil {
		http.Error(w, "JSON non valido", http.StatusBadRequest)
		return
	}

	query := "INSERT INTO comuni (nome, abitanti) VALUES ($1, $2)"
	err = env.db.QueryRow(query, comune.Nome, comune.Abitanti).Scan(&comune.Nome)
	if err != nil {
		http.Error(w, "errore nel salvataggio dei dati", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comune)
}

func (env *Env) getNewsByComune(w http.ResponseWriter, r *http.Request) {
	comune := r.PathValue("comune")
	if comune == "" {
		http.Error(w, "Comune mancante nell'URL", http.StatusBadRequest)
		return
	}

	query := "SELECT * FROM notizie WHERE comune = $1"
	rows, err := env.db.Query(query, comune)
	if err != nil {
		http.Error(w, "Errore interno del database", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	notizie := make([]Notizia, 0)

	for rows.Next() {
		var news Notizia

		err := rows.Scan(&news.Comune, &news.Contenuto, &news.Data_pubblicazione, &news.Giornale, &news.Tipologia, &news.Titolo)

		if err != nil {
			http.Error(w, "Errore nel database", http.StatusInternalServerError)
			return
		}

		notizie = append(notizie, news)
	}

	err = rows.Err()
	if err != nil {
		http.Error(w, "Errore interno del database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(notizie)
}

func (env *Env) getNewsByTipologia(w http.ResponseWriter, r *http.Request) {
	tipologia := r.PathValue("tipologia")
	if tipologia == "" {
		http.Error(w, "Tipologia mancante nell'URL", http.StatusBadRequest)
		return
	}

	query := "SELECT * FROM notizie WHERE tipologia = $1"
	rows, err := env.db.Query(query, tipologia)
	if err != nil {
		http.Error(w, "Errore nella esecuzione della query", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	notizie := make([]Notizia, 0)

	for rows.Next() {
		var news Notizia
		err := rows.Scan(&news.Comune, &news.Contenuto, &news.Data_pubblicazione, &news.Giornale, &news.Tipologia, &news.Titolo)
		if err != nil {
			http.Error(w, "Errore durante la lettura dei dati", http.StatusInternalServerError)
			return
		}

		notizie = append(notizie, news)
	}

	err = rows.Err()
	if err != nil {
		http.Error(w, "Errore interno del database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(notizie)
}

func (env *Env) getNewsByTitolo(w http.ResponseWriter, r *http.Request) {

	titolo := r.PathValue("titolo")
	if titolo == "" {
		http.Error(w, "Titolo mancante nell'URL", http.StatusBadRequest)
		return
	}

	query := "SELECT * FROM notizie WHERE titolo = $1"

	var news Notizia
	err := env.db.QueryRow(query, titolo).Scan(&news.Comune, &news.Contenuto, &news.Data_pubblicazione, &news.Giornale, &news.Tipologia, &news.Titolo) //qui devo sistemare il contenuto di scan mappando ogni campo di news con le relative colonne in ordine
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Notizia non trovata", http.StatusNotFound) // 404
			return
		}
		http.Error(w, "Errore nella connesione al db", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(news)

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

	insertQuery := `INSERT INTO notizie (giornale, titolo, comune, contenuto, tipologia, data) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	err = env.db.QueryRow(insertQuery, req.Giornale, req.Titolo, req.Comune, req.Contenuto, req.Tipologia, req.Data_pubblicazione).Scan(&req.Comune)
	if err != nil {
		http.Error(w, "Errore durante il salvataggio della notizia", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)

}
