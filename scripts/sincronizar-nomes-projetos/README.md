# sincronizar-nomes-projetos

Lista as configurações de projeto do ApiManager (`GET /projectsetting`), encontra o projeto correspondente no Zoho CRM (função `listarprojetos`) e grava o nome no `ProjectName` (`PUT /projectsetting/{projectId}/name`).

PRD: [prds/001-sincronizar-nomes-projetos.md](../../prds/001-sincronizar-nomes-projetos.md)

- **Linguagem:** Go 1.26 (só biblioteca padrão)
- **Dependências:** nenhuma além do Go

## Regra do nome

O `projectId` é dividido no primeiro `-`: a primeira parte é o id do projeto no CRM e o restante vira sufixo entre parênteses.

| `projectId` | Nome gravado |
| --- | --- |
| `4918426000054386272` | `Vivo SVA` |
| `4918426000054386272-IA-DEV` | `Vivo SVA (IA-DEV)` |

- Configurações que já têm nome são mantidas, a menos que se use `-forcar`.
- Configurações cujo id base não está no CRM (ou não é numérico) nunca são alteradas.

## Variáveis de ambiente

Lidas do ambiente ou, quando não definidas, do `.env` na raiz do repositório (modelo em [.env.example](../../.env.example)).

| Variável | Uso |
| --- | --- |
| `ZOHO_ZAPIKEY` | Chave da API da função `listarprojetos` no CRM. |
| `APIMANAGER_ACCESS_TOKEN` | Token do header `access-token` das rotas administrativas do ApiManager (o mesmo nos dois ambientes). |
| `APIMANAGER_URL_TESTE` | URL base do ApiManager de teste (`https://localhost:7119`). |
| `APIMANAGER_URL_PROD` | URL base do ApiManager de produção (`https://apimanager.macrolabs.app`). |

## Parâmetros

| Parâmetro | Descrição |
| --- | --- |
| `-ambiente teste\|prod` | Ambiente do ApiManager, usado na listagem e na gravação. Padrão: `teste`. |
| `-id <id>` | Processa só as configurações desse id base (com e sem sufixo). |
| `-dry-run` | Mostra o que seria feito, sem gravar no ApiManager. |
| `-forcar` | Sobrescreve também as configurações que já têm nome. |

## Como executar

Copie `.env.example` para `.env` na raiz do repositório e preencha. Teste primeiro com o projeto de teste (`4918426001337478951`, "Teste").

### Local (`-ambiente teste`, `https://localhost:7119`)

Teste é o ambiente padrão, então `-ambiente teste` pode ser omitido; aqui ele aparece explícito.

```bash
go run . -ambiente teste -dry-run                          # mostra o que mudaria
go run . -ambiente teste -dry-run -forcar                  # mostra o que mudaria sobrescrevendo nomes existentes
go run . -ambiente teste -id 4918426001337478951           # só o projeto de teste
go run . -ambiente teste -forcar -id 4918426001337478951   # só o projeto de teste, sobrescrevendo o nome
go run . -ambiente teste                                   # todas as configurações
go run . -ambiente teste -forcar                           # todas, sobrescrevendo nomes existentes
```

### Produção (`-ambiente prod`, `https://apimanager.macrolabs.app`)

```bash
go run . -ambiente prod -dry-run                           # mostra o que mudaria
go run . -ambiente prod -dry-run -forcar                   # mostra o que mudaria sobrescrevendo nomes existentes
go run . -ambiente prod -id 4918426001337478951            # só o projeto de teste
go run . -ambiente prod -forcar -id 4918426001337478951    # só o projeto de teste, sobrescrevendo o nome
go run . -ambiente prod                                    # todas as configurações
go run . -ambiente prod -forcar                            # todas, sobrescrevendo nomes existentes
```

### Testes automatizados

```bash
go test ./...                                              # testes da regra do nome
```

O ambiente de teste usa o certificado de desenvolvimento do .NET, que precisa estar confiável (`dotnet dev-certs https --trust`).

## Saída

Uma linha por configuração e um resumo no final:

- `[ok]`: nome gravado (com `-dry-run`, `[atualizaria]`);
- `[mantida]`: já tinha nome e `-forcar` não foi usado;
- `[sem projeto no CRM]`: id base não encontrado no CRM;
- `[falha]`: erro na gravação, com status e corpo.

O código de saída é diferente de zero se alguma gravação falhar.
