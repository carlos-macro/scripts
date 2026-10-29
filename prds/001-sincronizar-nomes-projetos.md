# PRD 001 — Sincronizar nomes dos projetos no ApiManager

**Status:** implementado · **Script:** [scripts/sincronizar-nomes-projetos](../scripts/sincronizar-nomes-projetos/)

## Problema

Os nomes dos projetos são mantidos no Zoho CRM, no módulo `Projetos`. A configuração de cada projeto no ApiManager (`ProjectSetting`) tem um campo `ProjectName` que não acompanha o CRM e fica vazio ou desatualizado.

## Objetivo

Um script que lê todos os projetos do CRM e grava o nome de cada um no `ProjectName` da configuração correspondente no ApiManager.

## Fontes

### Leitura: Zoho CRM

Função `standalone.listarProjetos` (código em `zoho-functions/CRM/Geral/Listar Projetos.deluge`).

```text
GET https://www.zohoapis.com/crm/v7/functions/listarprojetos/actions/execute?auth_type=apikey&zapikey=<ZOHO_ZAPIKEY>
```

O Zoho devolve o retorno da função como texto em `details.output`:

```json
{
  "code": "success",
  "details": { "output": "{\"data\":[{\"id\":\"4918426001337478951\",\"name\":\"Teste\"}],\"more_records\":false}" }
}
```

- `data[].id`: id do registro no CRM, que também é o id do projeto no ApiManager.
- `more_records`: `true` indica que a função atingiu o limite de 2.000 registros e há projetos que não vieram.

### Escrita: ApiManager

```text
PUT <APIMANAGER_URL>/projectsetting/{id}/name
access-token: <APIMANAGER_ACCESS_TOKEN>
Content-Type: application/json

{ "projectName": "Nome do Projeto" }
```

Responde `2xx` quando grava e `404` quando o projeto não tem configuração no ApiManager.

#### Ambientes

O ApiManager tem dois ambientes, cada um com sua URL base:

| Ambiente | Variável | Valor |
| --- | --- | --- |
| teste | `APIMANAGER_URL_TESTE` | `https://localhost:7119` (ApiManager rodando localmente) |
| prod | `APIMANAGER_URL_PROD` | `https://apimanager.macrolabs.app` |

- As URLs ficam no `.env` (modelo no `.env.example`), não no código. O `access-token` é o mesmo nos dois ambientes.
- A opção `-ambiente teste|prod` escolhe qual URL usar. O padrão é `teste`, para que escrever em produção exija pedir `-ambiente prod` explicitamente.
- Se a variável do ambiente escolhido não estiver definida, o script falha antes de ler o CRM.
- O script exibe no início o ambiente e a URL que vai usar.
- O ambiente de teste usa o certificado de desenvolvimento do .NET, que precisa estar confiável na máquina (`dotnet dev-certs https --trust`). O script não desativa a verificação de TLS.

## Requisitos

1. Ler a lista de projetos do CRM e falhar com mensagem clara se a chamada ou o formato da resposta forem inválidos.
2. Para cada projeto, chamar o `PUT` com o nome vindo do CRM.
3. `404` não é erro: o projeto é contado como "sem configuração" e o script segue.
4. Qualquer outra resposta fora de `2xx` é registrada como falha (com status e corpo) e o script segue para o próximo.
5. Ao final, exibir um resumo (atualizados, sem configuração, falhas). Sair com código diferente de zero se houver falha.
6. Avisar se `more_records` vier `true`.
7. Opção `-id` para processar um único projeto (usada nos testes com o projeto `4918426001337478951`, "Teste").
8. Opção `-dry-run` para listar o que seria enviado sem chamar o ApiManager.
9. Opção `-ambiente teste|prod` (padrão `teste`) para escolher a URL do ApiManager; ver "Ambientes".

## Segurança

- A chave do Zoho e o token do ApiManager vêm das variáveis de ambiente `ZOHO_ZAPIKEY` e `APIMANAGER_ACCESS_TOKEN`; nunca ficam no código nem no repositório.
- Variáveis não definidas no ambiente são lidas do `.env` na raiz do repositório (a primeira pasta acima do diretório atual que contém `.git`), no formato `CHAVE=valor`, com linhas vazias e `#` ignoradas. Variáveis já definidas no ambiente têm prioridade. O `.env` não é versionado; o modelo é o `.env.example`.
- Com `-ambiente prod`, o script escreve em produção. A execução completa em produção (sem `-id`) só acontece com autorização explícita.

## Fora do escopo

- Criar configuração no ApiManager para projetos que não têm.
- Remover ou renomear projetos no CRM.
- Paginação além do limite de 2.000 registros da função do CRM.
