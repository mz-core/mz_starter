# mz_starter

Camada de boot, configuracao e instalacao para subir um servidor FiveM usando `mz_core`.

## O que este repositorio e
- `mz_starter` organiza o `server.cfg`, os arquivos em `cfg/` e o fluxo basico de dependencias do ambiente.
- Ele existe para preparar e iniciar o servidor, nao para concentrar a logica do framework.

## O que este repositorio nao e
- `mz_starter` nao e o framework.
- A logica de dominio do servidor fica no `mz_core`: player, sessao, orgs, contas, inventario, veiculos e demais regras do core.

## Relacao com recursos oficiais
- O instalador sincroniza os recursos stock necessarios para o boot atual a partir do repositorio oficial `citizenfx/cfx-server-data`.
- Hoje ele traz `mapmanager`, `spawnmanager`, `sessionmanager`, `baseevents`, `chat` e os builders `yarn`/`webpack` usados pelo chat, mantendo o restante da base stock fora do escopo deste repositorio.

## O que o instalador faz hoje
- baixa `oxmysql`
- baixa `ox_lib`
- sincroniza os recursos CFX base usados no `cfg/resources.cfg`
- clona ou atualiza `mz_core`
- clona ou atualiza `mz_notify`
- clona ou atualiza `mz_sync`
- clona ou atualiza os resources MZ listados no boot atual
- clona ou atualiza `pma-voice`

## O que o instalador nao faz hoje
- nao instala FXServer artifacts
- nao provisiona a base stock completa do FiveM
- nao substitui configuracoes locais do servidor

## Instalacao rapida
1. Prepare os FXServer artifacts do ambiente.
2. Configure o banco em `cfg/database.cfg`.
3. Configure a license key em `cfg/base.cfg`.
4. Rode `tools/install_deps.bat` ou `tools/install_deps.sh`.
5. Inicie o servidor a partir da raiz deste projeto.

## Documentacao
- [Instalacao](docs/install.md)
- [Estrutura do ambiente](docs/structure.md)
