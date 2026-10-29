// Sincroniza o ProjectName das configurações do ApiManager com os nomes dos
// projetos do Zoho CRM. Ver README.md e prds/001-sincronizar-nomes-projetos.md.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const listarProjetosURL = "https://www.zohoapis.com/crm/v7/functions/listarprojetos/actions/execute"

// Variável com a URL base do ApiManager de cada ambiente.
var urlPorAmbiente = map[string]string{
	"teste": "APIMANAGER_URL_TESTE",
	"prod":  "APIMANAGER_URL_PROD",
}

var client = &http.Client{Timeout: 60 * time.Second}

type projeto struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
	id := flag.String("id", "", "processa só o projeto com este id")
	dryRun := flag.Bool("dry-run", false, "lista o que seria enviado sem chamar o ApiManager")
	ambiente := flag.String("ambiente", "teste", "ambiente do ApiManager: teste ou prod")
	flag.Parse()

	if err := run(*id, *dryRun, *ambiente); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run(id string, dryRun bool, ambiente string) error {
	if err := carregarEnv(); err != nil {
		return err
	}

	variavelURL, ok := urlPorAmbiente[ambiente]
	if !ok {
		return fmt.Errorf("ambiente %q inválido: use teste ou prod", ambiente)
	}
	baseURL := strings.TrimRight(os.Getenv(variavelURL), "/")
	if baseURL == "" {
		return fmt.Errorf("defina %s no ambiente ou no .env", variavelURL)
	}
	fmt.Printf("ambiente %s: %s\n", ambiente, baseURL)

	zapikey := os.Getenv("ZOHO_ZAPIKEY")
	if zapikey == "" {
		return errors.New("defina ZOHO_ZAPIKEY no ambiente ou no .env")
	}
	token := os.Getenv("APIMANAGER_ACCESS_TOKEN")
	if token == "" && !dryRun {
		return errors.New("defina APIMANAGER_ACCESS_TOKEN no ambiente ou no .env")
	}

	projetos, maisRegistros, err := listarProjetos(zapikey)
	if err != nil {
		return fmt.Errorf("listar projetos no CRM: %w", err)
	}
	fmt.Printf("%d projetos lidos do CRM\n", len(projetos))
	if maisRegistros {
		fmt.Println("aviso: o CRM indicou more_records=true; há projetos além do limite de 2.000 que não foram lidos")
	}

	if id != "" {
		projetos = filtrar(projetos, id)
		if len(projetos) == 0 {
			return fmt.Errorf("projeto %s não encontrado no CRM", id)
		}
	}

	var atualizados, semConfiguracao, falhas int
	for _, p := range projetos {
		if dryRun {
			fmt.Printf("[dry-run] %s -> %q\n", p.ID, p.Name)
			continue
		}
		status, corpo, err := atualizarNome(baseURL, token, p)
		switch {
		case err != nil:
			falhas++
			fmt.Printf("[falha] %s %q: %v\n", p.ID, p.Name, err)
		case status == http.StatusNotFound:
			semConfiguracao++
			fmt.Printf("[sem configuração] %s %q\n", p.ID, p.Name)
		case status >= 200 && status < 300:
			atualizados++
			fmt.Printf("[ok] %s %q\n", p.ID, p.Name)
		default:
			falhas++
			fmt.Printf("[falha] %s %q: HTTP %d %s\n", p.ID, p.Name, status, corpo)
		}
	}

	if dryRun {
		return nil
	}
	fmt.Printf("\nresumo: %d atualizados, %d sem configuração, %d falhas\n", atualizados, semConfiguracao, falhas)
	if falhas > 0 {
		return fmt.Errorf("%d projetos falharam", falhas)
	}
	return nil
}

// listarProjetos chama a função listarprojetos do CRM, que devolve
// {"data":[{"id","name"}],"more_records"} como texto em details.output.
func listarProjetos(zapikey string) ([]projeto, bool, error) {
	q := url.Values{"auth_type": {"apikey"}, "zapikey": {zapikey}}
	resp, err := client.Get(listarProjetosURL + "?" + q.Encode())
	if err != nil {
		// A URL leva a zapikey; não a repassa na mensagem de erro.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, false, err
	}
	defer resp.Body.Close()
	corpo, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("HTTP %d %s", resp.StatusCode, corpo)
	}

	var envelope struct {
		Code    string `json:"code"`
		Details struct {
			Output string `json:"output"`
		} `json:"details"`
	}
	if err := json.Unmarshal(corpo, &envelope); err != nil {
		return nil, false, fmt.Errorf("resposta inválida: %w", err)
	}
	if envelope.Code != "success" {
		return nil, false, fmt.Errorf("código %q: %s", envelope.Code, corpo)
	}

	var saida struct {
		Data        []projeto `json:"data"`
		MoreRecords bool      `json:"more_records"`
		Error       string    `json:"error"`
	}
	if err := json.Unmarshal([]byte(envelope.Details.Output), &saida); err != nil {
		return nil, false, fmt.Errorf("details.output inválido: %w", err)
	}
	if saida.Error != "" {
		return nil, false, errors.New(saida.Error)
	}
	return saida.Data, saida.MoreRecords, nil
}

func atualizarNome(baseURL, token string, p projeto) (int, string, error) {
	payload, err := json.Marshal(map[string]string{"projectName": p.Name})
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequest(http.MethodPut, baseURL+"/projectsetting/"+url.PathEscape(p.ID)+"/name", bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "*/*")
	req.Header.Set("access-token", token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(corpo)), nil
}

func filtrar(projetos []projeto, id string) []projeto {
	for _, p := range projetos {
		if p.ID == id {
			return []projeto{p}
		}
	}
	return nil
}
