# === STAGE 1: Compilazione del binario ===
FROM golang:1.22-alpine AS builder

# Impostiamo la cartella di lavoro dentro il container
WORKDIR /app

# Copiamo i file di gestione delle dipendenze per fare il caching
COPY go.mod go.sum ./
RUN go mod download

# Copiamo tutto il resto del codice sorgente
COPY . .

# Compiliamo il binario statico (CGO_ENABLED=0 lo rende indipendente dalle librerie di sistema)
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service .

# === STAGE 2: Immagine di esecuzione minimale ===
FROM alpine:latest

WORKDIR /root/

# Copiamo solo il file eseguibile compilato dallo stage precedente
COPY --from=builder /app/auth-service .

# Esponiamo la porta su cui gira il tuo server Go (es: 8080)
EXPOSE 8081

# Comando per avviare il microservizio
CMD ["./provincia-varese"]