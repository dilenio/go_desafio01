package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	serverURL  = "http://localhost:8080/cotacao"
	timeout    = 300 * time.Millisecond
	outputFile = "cotacao.txt"
)

type Cotacao struct {
	Bid string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Fatal("Tempo de execução excedido ao fazer a requisição.")
		} else {
			log.Fatal(err)
		}
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode == http.StatusBadRequest {
		log.Fatal(string(body))
	} else if res.StatusCode == http.StatusRequestTimeout {
		log.Fatal(string(body))
	}

	var cotacao Cotacao
	err = json.Unmarshal(body, &cotacao)
	if err != nil {
		log.Fatal("Erro ao fazer a conversão do JSON", err)
	}

	err = saveToFile(cotacao.Bid)
	if err != nil {
		log.Fatal("Erro ao salvar cotação no arquivo", err)
	}

	fmt.Printf("Cotação do dólar: %s\n", cotacao.Bid)
}

func saveToFile(bid string) error {
	content := fmt.Sprintf("Dólar: %s", bid)
	return os.WriteFile(outputFile, []byte(content), 0644)
}
