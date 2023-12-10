package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	apiURL      = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	dbPath      = "cotacoes.db"
	timeoutAPI  = 200 * time.Millisecond
	timeoutSave = 10 * time.Millisecond
)

type Cotacao struct {
	Bid string `json:"bid"`
}

type CotacaoResponse struct {
	USDBRL Cotacao `json:"USDBRL"`
}

func main() {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cotacoes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bid REAL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeoutAPI)
		defer cancel()

		cotacao, err := getExchange(ctx)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log.Println("Timeout ao obter cotação")
				http.Error(w, "Timeout ao obter cotação", http.StatusRequestTimeout)
				return
			}
			log.Println(err)
			http.Error(w, "Erro ao obter cotação", http.StatusBadRequest)
			return
		}
		
		valor, err := strconv.ParseFloat(cotacao.Bid, 64)
		if err != nil {
			log.Println("Erro ao converter dado")
		}

		err = saveExchange(ctx, db, valor)
		if err != nil {
			log.Printf("Erro ao salvar cotação no banco de dados: %v", err)
			http.Error(w, "Erro ao salvar cotação", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"bid": cotacao.Bid,
		})
	})

	http.ListenAndServe(":8080", nil)

}

func getExchange(ctx context.Context) (Cotacao, error) {
	client := http.Client{
		Timeout: timeoutAPI,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Cotacao{}, err
	}

	res, err := client.Do(req)
	if err != nil {
		return Cotacao{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Cotacao{}, err
	}

	var cotacaoResponse CotacaoResponse
	err = json.Unmarshal(body, &cotacaoResponse)
	if err != nil {
		return Cotacao{}, err
	}

	return cotacaoResponse.USDBRL, nil
}

func saveExchange(ctx context.Context, db *sql.DB, valor float64) error {
	ctx, cancel := context.WithTimeout(ctx, timeoutSave)
	defer cancel()

	_, err := db.ExecContext(ctx, "INSERT INTO cotacoes (bid) VALUES (?)", valor)
	return err
}