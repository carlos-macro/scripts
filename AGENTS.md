# Instruções para Agentes

Este repositório reúne scripts utilitários usados no trabalho com os projetos da Macro. É um repositório Git independente; as regras gerais de `../AGENTS.md` também se aplicam, e prevalece a regra mais restritiva.

## Idioma

Responda sempre em **português do Brasil**.

## Convenções

- Cada script fica em sua própria pasta, `scripts/<nome-do-script>/`, em kebab-case. Os scripts podem ser em linguagens diferentes; arquivos, dependências e configurações de um script ficam só na pasta dele.
- Cada pasta de script tem um `README.md` com o que o script faz, a linguagem, as dependências, como executar e quais parâmetros recebe.
- Todo script nasce de um PRD em `prds/NNN-<nome-do-script>.md` (numeração sequencial). Escreva ou atualize o PRD, **pare e peça para o desenvolvedor ler**; só implemente depois da aprovação explícita dele. Mantenha o PRD coerente com o comportamento do script.
- Ao criar ou alterar um script, atualize o `README.md` dele e a tabela de scripts do `README.md` da raiz.
- Não coloque segredos (tokens, senhas, credenciais) nos scripts; leia-os de variáveis de ambiente ou arquivos fora do versionamento.
- Scripts que escrevem em sistemas reais (Zoho CRM, bancos, APIs de produção) só são executados com autorização explícita.

## Git

- `commit`, `push` e demais ações listadas em `../AGENTS.md` exigem autorização explícita no momento.
- Os commits são do desenvolvedor. **Não se referencie como autor nem coautor**: nada de `Co-Authored-By`, "Generated with…", menção a Claude, IA ou agente em mensagens de commit, descrições de PR, comentários ou código.
- Não altere `git config`.
