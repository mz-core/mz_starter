# Instalacao

## Objetivo deste guia
Este documento descreve o contrato operacional atual do setup do `mz_starter`.

O foco aqui e deixar claro:
- o que precisa existir antes de rodar o instalador
- o que o instalador realmente faz
- o que continua fora do escopo deste repositorio

## Papel de cada parte no setup

### `mz_starter`
Responsavel pela camada de boot do projeto:
- `server.cfg`
- arquivos em `cfg/`
- scripts de instalacao de dependencias do projeto

### `mz_core`
Responsavel pela logica do framework:
- dominio de player e sessao
- orgs, contas, inventario, veiculos e logs
- regras e fluxos internos do core

### `cfx-server-data`
Responsavel pela base stock do servidor FiveM.

No estado atual deste projeto, o instalador usa o repositorio oficial `citizenfx/cfx-server-data` para sincronizar apenas os recursos stock necessarios ao boot atual:
- `mapmanager`
- `spawnmanager`
- `sessionmanager`

O restante da base stock continua fora do escopo do `mz_starter`.

## Pre-requisitos
Antes de rodar o instalador, o ambiente precisa ter:
- FXServer artifacts ja preparados
- Git disponivel
- acesso a internet para baixar dependencias
- `curl` e `unzip` disponiveis no Linux

## Fluxo recomendado de instalacao
1. Prepare o ambiente base do FiveM com seus artifacts.
2. Coloque o `mz_starter` como raiz do `server-data` do projeto.
3. Ajuste as configuracoes locais:
   - `cfg/database.cfg`
   - `cfg/base.cfg`
   - `cfg/permissions.cfg`, se necessario
4. Rode o instalador de dependencias do projeto.
5. Inicie o servidor usando o `server.cfg` deste repositorio.

## Como rodar o instalador

### Windows
1. Abra a pasta do `mz_starter`.
2. Rode `tools\install_deps.bat`.

### Linux
1. Abra o terminal na pasta do `mz_starter`.
2. Rode `bash tools/install_deps.sh`.

## O que o instalador faz hoje
- baixa a release estavel do `oxmysql`
- baixa a release estavel do `ox_lib`
- sincroniza `mapmanager`, `spawnmanager` e `sessionmanager` do repositorio oficial `citizenfx/cfx-server-data`
- clona ou atualiza `mz_core`
- clona ou atualiza `mz_notify`
- clona ou atualiza `pma-voice`

## O que o instalador nao faz hoje
- nao instala FXServer artifacts
- nao baixa a base stock completa do FiveM
- nao cria configs locais do servidor

## O que precisa ficar entendido sem ambiguidade
- O `mz_starter` nao substitui o `mz_core`.
- O `mz_starter` usa apenas uma parte do `cfx-server-data`, nao a base completa.
- O `mz_core` entra como recurso do framework dentro de `resources/[mz]/mz_core`.
- O `mz_notify` entra em `resources/[mz]/mz_notify`.
- O `pma-voice` entra em `resources/[som]/pma-voice`.
- Os recursos oficiais do boot atual entram em `resources/[managers]` e `resources/[system]`.

## Resultado esperado
Ao final desta etapa, o ambiente deve ter:
- o `server.cfg` e os `cfg/*.cfg` do `mz_starter`
- `oxmysql` em `resources/[ox]/oxmysql`
- `ox_lib` em `resources/[ox]/ox_lib`
- `mapmanager` em `resources/[managers]/mapmanager`
- `spawnmanager` em `resources/[managers]/spawnmanager`
- `sessionmanager` em `resources/[system]/sessionmanager`
- `pma-voice` em `resources/[som]/pma-voice`
- `mz_core` em `resources/[mz]/mz_core`
- `mz_notify` em `resources/[mz]/mz_notify`

Para a visao de fronteira entre os repositorios e a estrutura esperada do diretorio, veja [docs/structure.md](structure.md).
