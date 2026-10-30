// Grava a configuração de abandono (recovery) dos projetos listados no arquivo
// JSON nas configurações do ApiManager. Ver README.md e
// prds/002-configurar-abandono-projetos.md.
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

// Máximo aceito pela listagem de configurações do ApiManager.
const tamanhoPagina = 200

// Variável com a URL base do ApiManager de cada ambiente.
var urlPorAmbiente = map[string]string{
	"teste": "APIMANAGER_URL_TESTE",
	"prod":  "APIMANAGER_URL_PROD",
}

var client = &http.Client{Timeout: 60 * time.Second}

// entrada é um item de "projetos" no arquivo. Body é enviado ao PUT como está.
type entrada struct {
	ProjectID string          `json:"projectId"`
	Body      json.RawMessage `json:"body"`
	recovery  recovery
}

type recovery struct {
	Enable         *bool `json:"enable"`
	Minutes        *int  `json:"minutes"`
	IgnoreChatbot  *bool `json:"ignoreChatbot"`
	ChatbotMinutes *int  `json:"chatbotMinutes"`
	MaxHours       *int  `json:"maxHours"`
}

type configuracao struct {
	ProjectID       string `json:"projectId"`
	RecoveryEnabled *bool  `json:"recoveryEnabled"`
	RecoveryMinutes *int   `json:"recoveryMinutes"`
}

type decisao int

const (
	semConfiguracao decisao = iota
	mantida
	atualizar
)

func main() {
	id := flag.String("id", "", "processa só a entrada com este projectId exato")
	dryRun := flag.Bool("dry-run", false, "mostra o que seria feito sem gravar no ApiManager")
	ambiente := flag.String("ambiente", "teste", "ambiente do ApiManager: teste ou prod")
	forcar := flag.Bool("forcar", false, "sobrescreve também as configurações que já têm abandono configurado")
	arquivo := flag.String("arquivo", "recovery-settings-projetos.json", "arquivo JSON com os projetos e a configuração de abandono")
	flag.Parse()

	if err := run(*id, *dryRun, *ambiente, *forcar, *arquivo); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run(id string, dryRun bool, ambiente string, forcar bool, arquivo string) error {
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
	token := os.Getenv("APIMANAGER_ACCESS_TOKEN")
	if token == "" {
		return errors.New("defina APIMANAGER_ACCESS_TOKEN no ambiente ou no .env")
	}
	fmt.Printf("ambiente %s: %s\n", ambiente, baseURL)
	fmt.Printf("arquivo: %s\n", arquivo)

	conteudo, err := os.ReadFile(arquivo)
	if err != nil {
		return err
	}
	entradas, err := lerEntradas(conteudo)
	if err != nil {
		return fmt.Errorf("%s: %w", arquivo, err)
	}
	fmt.Printf("%d entradas lidas do arquivo\n", len(entradas))

	if id != "" {
		entradas = filtrar(entradas, id)
		if len(entradas) == 0 {
			return fmt.Errorf("nenhuma entrada com projectId %s no arquivo", id)
		}
	}

	configuracoes, err := listarConfiguracoes(baseURL, token)
	if err != nil {
		return fmt.Errorf("listar configurações no ApiManager: %w", err)
	}
	fmt.Printf("%d configurações lidas do ApiManager\n", len(configuracoes))
	porID := make(map[string]configuracao, len(configuracoes))
	for _, c := range configuracoes {
		porID[c.ProjectID] = c
	}

	var atualizadas, mantidas, semConfig, falhas int
	for _, e := range entradas {
		c, existe := porID[e.ProjectID]
		linha := e.ProjectID + " " + e.recovery.resumo()
		if existe && configurado(c) {
			linha += fmt.Sprintf(" (atual: enable=%t minutes=%d)", *c.RecoveryEnabled, *c.RecoveryMinutes)
		}

		switch decidir(c, existe, forcar) {
		case semConfiguracao:
			semConfig++
			fmt.Printf("[sem configuração] %s\n", e.ProjectID)
			continue
		case mantida:
			mantidas++
			fmt.Printf("[mantida] %s\n", linha)
			continue
		}

		if dryRun {
			atualizadas++
			fmt.Printf("[atualizaria] %s\n", linha)
			continue
		}
		status, corpo, err := atualizarRecovery(baseURL, token, e.ProjectID, e.Body)
		switch {
		case err != nil:
			falhas++
			fmt.Printf("[falha] %s: %v\n", linha, err)
		case status >= 200 && status < 300:
			atualizadas++
			fmt.Printf("[ok] %s\n", linha)
		default:
			falhas++
			fmt.Printf("[falha] %s: HTTP %d %s\n", linha, status, corpo)
		}
	}

	verbo := "atualizadas"
	if dryRun {
		verbo = "a atualizar"
	}
	fmt.Printf("\nresumo: %d %s, %d mantidas, %d sem configuração, %d falhas\n", atualizadas, verbo, mantidas, semConfig, falhas)
	if falhas > 0 {
		return fmt.Errorf("%d configurações falharam", falhas)
	}
	return nil
}

// lerEntradas lê e valida o arquivo inteiro: projectId presente e único, e
// todos os campos do body presentes (chatbotMinutes pode ser null).
func lerEntradas(conteudo []byte) ([]entrada, error) {
	var arquivo struct {
		Projetos []entrada `json:"projetos"`
	}
	if err := json.Unmarshal(conteudo, &arquivo); err != nil {
		return nil, fmt.Errorf("JSON inválido: %w", err)
	}
	if len(arquivo.Projetos) == 0 {
		return nil, errors.New(`nenhuma entrada em "projetos"`)
	}

	vistos := make(map[string]bool, len(arquivo.Projetos))
	for i := range arquivo.Projetos {
		e := &arquivo.Projetos[i]
		onde := fmt.Sprintf("entrada %d", i+1)
		if e.ProjectID == "" {
			return nil, fmt.Errorf("%s: projectId ausente", onde)
		}
		onde += " (" + e.ProjectID + ")"
		if vistos[e.ProjectID] {
			return nil, fmt.Errorf("%s: projectId repetido", onde)
		}
		vistos[e.ProjectID] = true

		var campos map[string]json.RawMessage
		if err := json.Unmarshal(e.Body, &campos); err != nil || campos == nil {
			return nil, fmt.Errorf("%s: body ausente ou inválido", onde)
		}
		if err := json.Unmarshal(e.Body, &e.recovery); err != nil {
			return nil, fmt.Errorf("%s: body inválido: %w", onde, err)
		}
		if _, ok := campos["chatbotMinutes"]; !ok {
			return nil, fmt.Errorf("%s: body sem chatbotMinutes (use null para não definir)", onde)
		}
		r := e.recovery
		for campo, presente := range map[string]bool{
			"enable":        r.Enable != nil,
			"minutes":       r.Minutes != nil,
			"ignoreChatbot": r.IgnoreChatbot != nil,
			"maxHours":      r.MaxHours != nil,
		} {
			if !presente {
				return nil, fmt.Errorf("%s: body sem %s", onde, campo)
			}
		}
	}
	return arquivo.Projetos, nil
}

// decidir aplica a regra do PRD: sem configuração no ApiManager nunca grava;
// abandono já configurado só é sobrescrito com -forcar.
func decidir(c configuracao, existe, forcar bool) decisao {
	switch {
	case !existe:
		return semConfiguracao
	case configurado(c) && !forcar:
		return mantida
	default:
		return atualizar
	}
}

// configurado indica se a configuração já tem abandono configurado. Sem
// abandono, o ApiManager devolve recoveryEnabled=false e recoveryMinutes=0.
func configurado(c configuracao) bool {
	return *c.RecoveryEnabled || *c.RecoveryMinutes > 0
}

func (r recovery) resumo() string {
	chatbot := "-"
	if r.ChatbotMinutes != nil {
		chatbot = fmt.Sprint(*r.ChatbotMinutes)
	}
	return fmt.Sprintf("enable=%t minutes=%d ignoreChatbot=%t chatbotMinutes=%s maxHours=%d",
		*r.Enable, *r.Minutes, *r.IgnoreChatbot, chatbot, *r.MaxHours)
}

// listarConfiguracoes lê todas as páginas de GET /projectsetting. Falha se
// algum item vier sem recoveryEnabled ou recoveryMinutes, porque sem eles não
// dá para saber o que seria sobrescrito.
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
		for _, c := range saida.Items {
			if c.RecoveryEnabled == nil || c.RecoveryMinutes == nil {
				return nil, fmt.Errorf("página %d: configuração %q sem recoveryEnabled ou recoveryMinutes", pagina, c.ProjectID)
			}
		}
		todas = append(todas, saida.Items...)
		if len(saida.Items) == 0 || len(todas) >= saida.Total {
			return todas, nil
		}
	}
}

func atualizarRecovery(baseURL, token, projectID string, body json.RawMessage) (int, string, error) {
	req, err := http.NewRequest(http.MethodPut, baseURL+"/projectsetting/"+url.PathEscape(projectID)+"/recovery", bytes.NewReader(body))
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

// filtrar mantém a entrada com projectId exato igual a id.
func filtrar(entradas []entrada, id string) []entrada {
	for _, e := range entradas {
		if e.ProjectID == id {
			return []entrada{e}
		}
	}
	return nil
}
