# sincronizar-nomes-projetos

Lê todos os projetos do Zoho CRM (função `listarprojetos`) e grava o nome de cada um no `ProjectName` da configuração do projeto no ApiManager (`PUT /projectsetting/{id}/name`).

PRD: [prds/001-sincronizar-nomes-projetos.md](../../prds/001-sincronizar-nomes-projetos.md)

- **Linguagem:** Go 1.26 (só biblioteca padrão)
- **Dependências:** nenhuma além do Go

## Variáveis de ambiente

Lidas do ambiente ou, quando não definidas, do `.env` na raiz do repositório (modelo em [.env.example](../../.env.example)).

| Variável | Uso |
| --- | --- |
| `ZOHO_ZAPIKEY` | Chave da API da função `listarprojetos` no CRM. |
| `APIMANAGER_ACCESS_TOKEN` | Token do header `access-token` do ApiManager (o mesmo nos dois ambientes). Não é exigido com `-dry-run`. |
| `APIMANAGER_URL_TESTE` | URL base do ApiManager de teste (`https://localhost:7119`). |
| `APIMANAGER_URL_PROD` | URL base do ApiManager de produção (`https://apimanager.macrolabs.app`). |

## Parâmetros

| Parâmetro | Descrição |
| --- | --- |
| `-ambiente teste\|prod` | Ambiente do ApiManager. Padrão: `teste`. |
| `-id <id>` | Processa só o projeto com esse id. |
| `-dry-run` | Lista o que seria enviado, sem chamar o ApiManager. |

## Como executar

Copie `.env.example` para `.env` na raiz do repositório e preencha. Teste primeiro com o projeto de teste (`4918426001337478951`, "Teste").

```bash
go run . -dry-run                                   # só lista
go run . -id 4918426001337478951                    # projeto de teste no ApiManager local
go run . -ambiente prod -id 4918426001337478951     # projeto de teste em produção
go run . -ambiente prod                             # todos os projetos em produção
```

O ambiente de teste usa o certificado de desenvolvimento do .NET, que precisa estar confiável (`dotnet dev-certs https --trust`).

## Saída

Uma linha por projeto (`[ok]`, `[sem configuração]` para HTTP 404 ou `[falha]` com status e corpo) e um resumo no final. O código de saída é diferente de zero se algum projeto falhar.
