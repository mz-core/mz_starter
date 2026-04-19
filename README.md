# mz_starter

Camada de boot, configuração e instalação para subir um servidor FiveM usando `mz_core`.

## O que este repositório é
- `mz_starter` organiza o `server.cfg`, os arquivos em `cfg/` e o fluxo básico de dependências do ambiente.
- Ele existe para preparar e iniciar o servidor, não para concentrar a lógica do framework.

## O que este repositório não é
- `mz_starter` não é o framework.
- A lógica de domínio do servidor fica no `mz_core`: player, sessão, orgs, contas, inventário, veículos e demais regras do core.

## Relação com `cfx-server-data`
- Este repositório assume um ambiente de `server-data` compatível com FiveM.
- Os recursos stock usados no boot atual, como `mapmanager` e `spawnmanager`, não vêm neste repositório.
- Eles precisam existir no ambiente por meio de `cfx-server-data` ou estrutura equivalente já preparada.

## O que o instalador faz hoje
- baixa `oxmysql`
- baixa `ox_lib`
- clona ou atualiza `mz_core`

## O que o instalador não faz hoje
- não instala FXServer artifacts
- não baixa `cfx-server-data`
- não provisiona os recursos stock do FiveM

## Instalação rápida
1. Prepare um ambiente FiveM com estrutura de `server-data` e recursos stock disponíveis.
2. Configure o banco em `cfg/database.cfg`.
3. Configure a license key em `cfg/base.cfg`.
4. Rode `tools/install_deps.bat` ou `tools/install_deps.sh`.
5. Inicie o servidor a partir da raiz deste projeto.

## Documentação
- [Instalação](docs/install.md)
- [Estrutura do ambiente](docs/structure.md)
