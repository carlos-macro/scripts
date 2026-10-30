package main

import (
	"os"
	"strings"
	"testing"
)

func TestLerEntradas(t *testing.T) {
	casos := []struct {
		nome  string
		json  string
		erro  string
		total int
	}{
		{"válido", `{"projetos":[
			{"projectId":"1","origem":"ignorado","body":{"enable":true,"minutes":20,"ignoreChatbot":false,"chatbotMinutes":null,"maxHours":24}},
			{"projectId":"1-VOZ","body":{"enable":true,"minutes":10,"ignoreChatbot":false,"chatbotMinutes":20,"maxHours":24}}]}`, "", 2},
		{"JSON inválido", `{"projetos":[`, "JSON inválido", 0},
		{"vazio", `{"projetos":[]}`, "nenhuma entrada", 0},
		{"sem projectId", `{"projetos":[{"body":{"enable":true,"minutes":1,"ignoreChatbot":false,"chatbotMinutes":null,"maxHours":4}}]}`, "projectId ausente", 0},
		{"repetido", `{"projetos":[
			{"projectId":"1","body":{"enable":true,"minutes":1,"ignoreChatbot":false,"chatbotMinutes":null,"maxHours":4}},
			{"projectId":"1","body":{"enable":true,"minutes":1,"ignoreChatbot":false,"chatbotMinutes":null,"maxHours":4}}]}`, "repetido", 0},
		{"sem body", `{"projetos":[{"projectId":"1"}]}`, "body ausente", 0},
		{"sem chatbotMinutes", `{"projetos":[{"projectId":"1","body":{"enable":true,"minutes":1,"ignoreChatbot":false,"maxHours":4}}]}`, "chatbotMinutes", 0},
		{"sem minutes", `{"projetos":[{"projectId":"1","body":{"enable":true,"ignoreChatbot":false,"chatbotMinutes":null,"maxHours":4}}]}`, "sem minutes", 0},
		{"enable null", `{"projetos":[{"projectId":"1","body":{"enable":null,"minutes":1,"ignoreChatbot":false,"chatbotMinutes":null,"maxHours":4}}]}`, "sem enable", 0},
		{"tipo errado", `{"projetos":[{"projectId":"1","body":{"enable":"sim","minutes":1,"ignoreChatbot":false,"chatbotMinutes":null,"maxHours":4}}]}`, "body inválido", 0},
	}
	for _, c := range casos {
		entradas, err := lerEntradas([]byte(c.json))
		if c.erro == "" {
			if err != nil {
				t.Errorf("%s: erro inesperado: %v", c.nome, err)
			} else if len(entradas) != c.total {
				t.Errorf("%s: %d entradas, esperado %d", c.nome, len(entradas), c.total)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.erro) {
			t.Errorf("%s: erro = %v, esperado contendo %q", c.nome, err, c.erro)
		}
	}
}

func TestDecidir(t *testing.T) {
	sim, nao := true, false
	zero, vinte := 0, 20
	semAbandono := configuracao{ProjectID: "1", RecoveryEnabled: &nao, RecoveryMinutes: &zero}
	habilitado := configuracao{ProjectID: "1", RecoveryEnabled: &sim, RecoveryMinutes: &vinte}
	desabilitadoComMinutos := configuracao{ProjectID: "1", RecoveryEnabled: &nao, RecoveryMinutes: &vinte}

	casos := []struct {
		nome     string
		c        configuracao
		existe   bool
		forcar   bool
		esperado decisao
	}{
		{"não existe", configuracao{}, false, false, semConfiguracao},
		{"não existe, com -forcar", configuracao{}, false, true, semConfiguracao},
		{"sem abandono", semAbandono, true, false, atualizar},
		{"habilitado", habilitado, true, false, mantida},
		{"habilitado, com -forcar", habilitado, true, true, atualizar},
		{"desabilitado com minutos", desabilitadoComMinutos, true, false, mantida},
	}
	for _, c := range casos {
		if d := decidir(c.c, c.existe, c.forcar); d != c.esperado {
			t.Errorf("%s: decisão %d, esperado %d", c.nome, d, c.esperado)
		}
	}
}

// O arquivo versionado precisa passar na validação.
func TestArquivoDoRepositorio(t *testing.T) {
	conteudo, err := os.ReadFile("recovery-settings-projetos.json")
	if err != nil {
		t.Fatal(err)
	}
	entradas, err := lerEntradas(conteudo)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtrar(entradas, "4918426001337478951")) != 1 {
		t.Error("projeto de teste 4918426001337478951 ausente do arquivo")
	}
}
