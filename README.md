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
- clona ou atualiza `screenshot-basic` em `resources/[standalone]`
- instala `pma-voice` v7.0.0 no commit `3379e45dcf7aaa8b7651317190c108095a121553`
- aplica e verifica o patch MZ que remove o ingresso em call channel solicitado pelo client, preservando o export server `setPlayerCall`

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

## screenshot-basic

O `screenshot-basic` e preparado como dependencia base/opcional em `resources/[standalone]/screenshot-basic` e iniciado em `cfg/resources.cfg` antes dos resources MZ.

Ele sera usado em uma fase futura do `mz_realestate` para fotos de imoveis. No estado atual, anuncios, placas e detalhes publicos continuam funcionando sem implementar captura de imagens.

Instalacao manual, se necessario:

```sh
mkdir -p "resources/[standalone]"
git clone https://github.com/citizenfx/screenshot-basic.git "resources/[standalone]/screenshot-basic"
```

## pma-voice

O `pma-voice` e instalado em `resources/[som]/pma-voice` antes de `mz_phone` e `mz_radio`. O pin e o patch fazem parte do contrato de seguranca do telefone: apenas resources server-side podem atribuir membership de chamada. O instalador aborta sem sobrescrever quando encontra uma copia sem `.git` ou alteracoes locais diferentes do patch esperado.

Nao atualize esse resource com `git pull` isolado. Para atualizar, mude tag, commit e patch juntos nos dois instaladores, valide o spoof de `pma-voice:setPlayerCall` e execute as regressões de `mz_phone` e `mz_radio`.
