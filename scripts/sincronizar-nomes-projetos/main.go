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

const (
	listarProjetosURL = "https://www.zohoapis.com/crm/v7/functions/listarprojetos/actions/execute"
	// Máximo aceito pela listagem de configurações do ApiManager.
	tamanhoPagina = 200
)

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

type configuracao struct {
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
}

func main() {
	id := flag.String("id", "", "processa só as configurações deste id base (com e sem sufixo)")
	dryRun := flag.Bool("dry-run", false, "mostra o que seria feito sem gravar no ApiManager")
	ambiente := flag.String("ambiente", "teste", "ambiente do ApiManager: teste ou prod")
	forcar := flag.Bool("forcar", false, "sobrescreve também as configurações que já têm nome")
	flag.Parse()

	if err := run(*id, *dryRun, *ambiente, *forcar); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run(id string, dryRun bool, ambiente string, forcar bool) error {
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
	zapikey := os.Getenv("ZOHO_ZAPIKEY")
	if zapikey == "" {
		return errors.New("defina ZOHO_ZAPIKEY no ambiente ou no .env")
	}
	token := os.Getenv("APIMANAGER_ACCESS_TOKEN")
	if token == "" {
		return errors.New("defina APIMANAGER_ACCESS_TOKEN no ambiente ou no .env")
	}
	fmt.Printf("ambiente %s: %s\n", ambiente, baseURL)

	projetos, maisRegistros, err := listarProjetos(zapikey)
	if err != nil {
		return fmt.Errorf("listar projetos no CRM: %w", err)
	}
	fmt.Printf("%d projetos lidos do CRM\n", len(projetos))
	if maisRegistros {
		fmt.Println("aviso: o CRM indicou more_records=true; há projetos além do limite de 2.000 que não foram lidos")
	}
	nomes := make(map[string]string, len(projetos))
	for _, p := range projetos {
		nomes[p.ID] = p.Name
	}

	configuracoes, err := listarConfiguracoes(baseURL, token)
	if err != nil {
		return fmt.Errorf("listar configurações no ApiManager: %w", err)
	}
	fmt.Printf("%d configurações lidas do ApiManager\n", len(configuracoes))

	if id != "" {
		configuracoes = filtrar(configuracoes, id)
		if len(configuracoes) == 0 {
			return fmt.Errorf("nenhuma configuração com id base %s no ApiManager", id)
		}
	}

	var atualizadas, mantidas, semProjeto, falhas int
	for _, c := range configuracoes {
		base, sufixo, ok := separarID(c.ProjectID)
		nomeCRM, existe := nomes[base]
		if !ok || !existe {
			semProjeto++
			fmt.Printf("[sem projeto no CRM] %s\n", c.ProjectID)
			continue
		}
		if c.ProjectName != "" && !forcar {
			mantidas++
			fmt.Printf("[mantida] %s %q\n", c.ProjectID, c.ProjectName)
			continue
		}

		nome := montarNome(nomeCRM, sufixo)
		if dryRun {
			atualizadas++
			fmt.Printf("[atualizaria] %s %q -> %q\n", c.ProjectID, c.ProjectName, nome)
			continue
		}
		status, corpo, err := atualizarNome(baseURL, token, c.ProjectID, nome)
		switch {
		case err != nil:
			falhas++
			fmt.Printf("[falha] %s %q: %v\n", c.ProjectID, nome, err)
		case status >= 200 && status < 300:
			atualizadas++
			fmt.Printf("[ok] %s %q -> %q\n", c.ProjectID, c.ProjectName, nome)
		default:
			falhas++
			fmt.Printf("[falha] %s %q: HTTP %d %s\n", c.ProjectID, nome, status, corpo)
		}
	}

	verbo := "atualizadas"
	if dryRun {
		verbo = "a atualizar"
	}
	fmt.Printf("\nresumo: %d %s, %d mantidas, %d sem projeto no CRM, %d falhas\n", atualizadas, verbo, mantidas, semProjeto, falhas)
	if falhas > 0 {
		return fmt.Errorf("%d configurações falharam", falhas)
	}
	return nil
}

// separarID divide o projectId no primeiro "-": o id base (id do projeto no
// CRM) e o sufixo. ok é false quando o id base não é numérico.
func separarID(projectID string) (base, sufixo string, ok bool) {
	base, sufixo, _ = strings.Cut(projectID, "-")
	if base == "" {
		return base, sufixo, false
	}
	for _, r := range base {
		if r < '0' || r > '9' {
			return base, sufixo, false
		}
	}
	return base, sufixo, true
}

// montarNome devolve o nome do CRM, com o sufixo entre parênteses quando houver.
func montarNome(nomeCRM, sufixo string) string {
	if sufixo == "" {
		return nomeCRM
	}
	return fmt.Sprintf("%s (%s)", nomeCRM, sufixo)
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

// listarConfiguracoes lê todas as páginas de GET /projectsetting.
func listarConfiguracoes(baseURL, token string) ([]configuracao, error) {
	var todas []configuracao
	for pagina := 1; ; pagina++ {
		q := url.Values{"page": {fmt.Sprint(pagina)}, "pageSize": {fmt.Sprint(tamanhoPagina)}}
		req, err := http.NewRequest(http.MethodGet, baseURL+"/projectsetting?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("accept", "*/*")
		req.Header.Set("access-token", token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		corpo, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("página %d: HTTP %d %s", pagina, resp.StatusCode, corpo)
		}

		var saida struct {
			Items []configuracao `json:"items"`
			Total int            `json:"total"`
		}
		if err := json.Unmarshal(corpo, &saida); err != nil {
			return nil, fmt.Errorf("página %d: resposta inválida: %w", pagina, err)
		}
		todas = append(todas, saida.Items...)
		if len(saida.Items) == 0 || len(todas) >= saida.Total {
			return todas, nil
		}
	}
}

func atualizarNome(baseURL, token, projectID, nome string) (int, string, error) {
	payload, err := json.Marshal(map[string]string{"projectName": nome})
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequest(http.MethodPut, baseURL+"/projectsetting/"+url.PathEscape(projectID)+"/name", bytes.NewReader(payload))
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

// filtrar mantém as configurações cujo id base é id.
func filtrar(configuracoes []configuracao, id string) []configuracao {
	var filtradas []configuracao
	for _, c := range configuracoes {
		if base, _, _ := separarID(c.ProjectID); base == id {
			filtradas = append(filtradas, c)
		}
	}
	return filtradas
}
