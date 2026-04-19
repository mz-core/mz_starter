# Instalação

## Objetivo deste guia
Este documento descreve o contrato operacional atual do setup do `mz_starter`.

O foco aqui é deixar claro:
- o que precisa existir antes de rodar o instalador
- o que o instalador realmente faz
- o que continua fora do escopo deste repositório

## Papel de cada parte no setup

### `mz_starter`
Responsável pela camada de boot do projeto:
- `server.cfg`
- arquivos em `cfg/`
- scripts de instalação de dependências do projeto

### `mz_core`
Responsável pela lógica do framework:
- domínio de player e sessão
- orgs, contas, inventário, veículos
- regras e fluxos internos do core

### `cfx-server-data`
Responsável pela base stock do ambiente FiveM.

No estado atual deste projeto, os recursos stock usados no boot, como `mapmanager` e `spawnmanager`, precisam já existir no ambiente por meio de `cfx-server-data` ou estrutura equivalente.

O instalador do `mz_starter` não baixa essa base.

## Pré-requisitos
Antes de rodar o instalador, o ambiente precisa ter:
- FXServer artifacts já preparados
- Git disponível
- acesso à internet para baixar dependências
- uma estrutura de `server-data` compatível com FiveM
- recursos stock do FiveM disponíveis, incluindo os recursos exigidos no boot atual

## Fluxo recomendado de instalação
1. Prepare o ambiente base do FiveM com seus artifacts.
2. Garanta a presença da base stock de `server-data`, via `cfx-server-data` ou estrutura equivalente.
3. Coloque o `mz_starter` como raiz do `server-data` do projeto.
4. Ajuste as configurações locais:
   - `cfg/database.cfg`
   - `cfg/base.cfg`
   - `cfg/permissions.cfg`, se necessário
5. Rode o instalador de dependências do projeto.
6. Inicie o servidor usando o `server.cfg` deste repositório.

## Como rodar o instalador

### Windows
1. Abra a pasta do `mz_starter`.
2. Rode `tools\install_deps.bat`.

### Linux
1. Abra o terminal na pasta do `mz_starter`.
2. Rode `bash tools/install_deps.sh`.

## O que o instalador faz hoje
- baixa a release estável do `oxmysql`
- baixa a release estável do `ox_lib`
- clona ou atualiza o `mz_core`

## O que o instalador não faz hoje
- não instala FXServer artifacts
- não baixa `cfx-server-data`
- não cria a base stock do servidor
- não valida automaticamente se os recursos stock exigidos pelo boot já existem

## O que precisa ficar entendido sem ambiguidade
- O `mz_starter` não substitui o `mz_core`.
- O `mz_starter` também não substitui o `cfx-server-data`.
- O `mz_core` entra como recurso do framework dentro de `resources/[mz]/mz_core`.
- Os recursos stock do boot atual continuam vindo da base `cfx-server-data` ou equivalente.
- Se essa base não existir, o ambiente fica incompleto mesmo que o instalador termine sem erro.

## Resultado esperado
Ao final desta etapa, o ambiente deve ter:
- o `server.cfg` e os `cfg/*.cfg` do `mz_starter`
- `oxmysql` em `resources/[ox]/oxmysql`
- `ox_lib` em `resources/[ox]/ox_lib`
- `mz_core` em `resources/[mz]/mz_core`
- recursos stock necessários ao boot já disponíveis no ambiente

Para a visão de fronteira entre os repositórios e a estrutura esperada do diretório, veja [docs/structure.md](structure.md).
