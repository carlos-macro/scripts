# configurar-abandono-projetos

Lê os projetos de [recovery-settings-projetos.json](recovery-settings-projetos.json), confere quais têm configuração no ApiManager (`GET /projectsetting`) e grava a configuração de abandono só nesses (`PUT /projectsetting/{projectId}/recovery`).

PRD: [prds/002-configurar-abandono-projetos.md](../../prds/002-configurar-abandono-projetos.md)

- **Linguagem:** Go 1.26 (só biblioteca padrão)
- **Dependências:** nenhuma além do Go

## Arquivo de entrada

```json
{
  "projetos": [
    {
      "projectId": "4918426001337478951",
      "body": { "enable": true, "minutes": 3, "ignoreChatbot": false, "chatbotMinutes": null, "maxHours": 24 }
    }
  ]
}
```

- `projectId`: id completo da configuração no ApiManager, com sufixo quando houver (`4918426000427538811-IA`).
- `body`: enviado ao `PUT` como está. Todos os campos são obrigatórios; `chatbotMinutes` pode ser `null`.
- Outros campos da entrada são ignorados.
- O arquivo inteiro é validado antes de qualquer chamada (JSON, campos obrigatórios, `projectId` repetido).

## Regra

- `projectId` que não existe no ApiManager não é alterado, nem com `-forcar`.
- Configuração que já tem abandono (`recoveryEnabled` `true` ou `recoverySettings` diferente de `null` na listagem) é mantida, a menos que se use `-forcar`.
- Se a listagem vier sem `recoveryEnabled` ou sem a chave `recoverySettings`, o script falha antes de gravar.
- As demais recebem o `body` do arquivo.

## Variáveis de ambiente

Lidas do ambiente ou, quando não definidas, do `.env` na raiz do repositório (modelo em [.env.example](../../.env.example)). São as mesmas do `sincronizar-nomes-projetos`.

| Variável | Uso |
| --- | --- |
| `APIMANAGER_ACCESS_TOKEN` | Token do header `access-token` das rotas administrativas do ApiManager (o mesmo nos dois ambientes). |
| `APIMANAGER_URL_TESTE` | URL base do ApiManager local (`https://localhost:7119`). |
| `APIMANAGER_URL_PROD` | URL base do ApiManager de produção (`https://apimanager.macrolabs.app`). |

## Parâmetros

| Parâmetro | Descrição |
| --- | --- |
| `-ambiente teste\|prod` | Ambiente do ApiManager, usado na listagem e na gravação. Padrão: `teste`. |
| `-id <projectId>` | Processa só a entrada com esse `projectId` exato (precisa estar no arquivo). |
| `-dry-run` | Mostra o que seria feito, sem gravar no ApiManager. |
| `-forcar` | Sobrescreve também as configurações que já têm abandono configurado. |
| `-arquivo <caminho>` | Outro arquivo de entrada. Padrão: `recovery-settings-projetos.json`. |

## Como executar

Copie `.env.example` para `.env` na raiz do repositório e preencha. Teste primeiro com o projeto de teste (`4918426001337478951`, "Teste"), que é a primeira entrada do arquivo.

### Local (`-ambiente teste`, `https://localhost:7119`)

Teste é o ambiente padrão, então `-ambiente teste` pode ser omitido; aqui ele aparece explícito.

```bash
go run . -ambiente teste -dry-run -id 4918426001337478951           # mostra o que mudaria no projeto de teste
go run . -ambiente teste -id 4918426001337478951                    # só o projeto de teste
go run . -ambiente teste -forcar -id 4918426001337478951            # só o projeto de teste, sobrescrevendo
go run . -ambiente teste -dry-run                                   # mostra o que mudaria
go run . -ambiente teste -dry-run -forcar                           # mostra o que mudaria sobrescrevendo
go run . -ambiente teste                                            # todas as entradas
go run . -ambiente teste -forcar                                    # todas, sobrescrevendo
```

### Produção (`-ambiente prod`, `https://apimanager.macrolabs.app`)

```bash
go run . -ambiente prod -dry-run -id 4918426001337478951            # mostra o que mudaria no projeto de teste
go run . -ambiente prod -id 4918426001337478951                     # só o projeto de teste
go run . -ambiente prod -forcar -id 4918426001337478951             # só o projeto de teste, sobrescrevendo
go run . -ambiente prod -dry-run                                    # mostra o que mudaria
go run . -ambiente prod -dry-run -forcar                            # mostra o que mudaria sobrescrevendo
go run . -ambiente prod                                             # todas as entradas
go run . -ambiente prod -forcar                                     # todas, sobrescrevendo
```

### Testes automatizados

```bash
go test ./...                                                       # validação do arquivo e regra de decisão
```

O ambiente local usa o certificado de desenvolvimento do .NET, que precisa estar confiável (`dotnet dev-certs https --trust`).

## Saída

Uma linha por entrada, com o `body` resumido e, quando já havia abandono configurado, o valor atual (`atual: enabled=... minutes=... ignoreChatbot=... chatbotMinutes=... maxHours=...`). Resumo no final:

- `[ok]`: configuração gravada (com `-dry-run`, `[atualizaria]`);
- `[mantida]`: já tinha abandono configurado e `-forcar` não foi usado;
- `[sem configuração]`: o `projectId` não existe no ApiManager do ambiente escolhido;
- `[falha]`: erro na gravação, com status e corpo.

O script não guarda backup: o valor anterior só aparece na saída. O código de saída é diferente de zero se alguma gravação falhar.
