# mz_starter

O Starter 2.0 planeja o ambiente e executa duas modalidades de instalacao apos confirmacao explicita: instalacao completa do MZ Core e instalacao parcial experimental. O planejamento e a instalacao estao documentados em [planejamento e catalogo](docs/planning.md). O Starter nao substitui os instaladores atuais e nao inicia o FXServer.

## Instalacao completa do MZ Core

Na interface, escolha Instalacao completa do MZ Core, informe um destino novo ou vazio e clique em Verificar componentes. O Starter consulta os releases reais dos repositorios oficiais `mz-core`, resolve os 51 componentes do perfil e mostra os bloqueios antes de instalar. O botao INSTALAR SERVIDOR baixa os componentes disponiveis, verifica SHA-256, inspeciona cada ZIP, extrai em staging privado e publica com rename sem sobrescrita.

O perfil publico atual resolve 26 GitHub Releases de resources MZ, 7 snapshots oficiais de dependencias (`oxmysql`, `ox_lib`, `pma-voice` com patch, `bob74_ipl`, `screenshot-basic`, `mapmanager` e `spawnmanager`), `baseevents` e 2 system resources. `mz_police` e publico e passou a ser baixado automaticamente. Recursos sem release publica utilizavel continuam identificados individualmente e podem ser atendidos por uma pasta local explicitamente autorizada. Releases privados de `mz_prison`, `mz_furniture` e `mz_admin` exigem token autorizado ou pacote local. `mz_banguard` permanece protegido e so pode vir de fonte local autorizada.

Os artifacts do FXServer sao separados do server-data. Na instalacao nova no Windows e no Linux, a opcao automatica baixa o build oficial recomendado e habilita `yarn`, `webpack`, `screenshot-basic` e `mz_feedback`. Se houver componentes obrigatorios sem distribuicao, o operador pode autorizar uma instalacao incompleta dos componentes disponiveis; o resultado e explicitamente marcado como incompleto e o servidor nao e declarado pronto. Licenca, banco, endpoints e permissoes continuam sendo responsabilidade do operador.

Ao final, o Starter cria `start-txadmin.cmd` e `start-txadmin.sh` dentro do server-data. Execute o script da sua plataforma para abrir o txAdmin; ele roda o FXServer sem `+exec server.cfg`, cria o `txData` nesse diretorio e imprime a URL do painel (padrao `http://127.0.0.1:40120`). No primeiro acesso, escolha a opcao de servidor existente e selecione o server-data recem-criado. Nao use o deploy de um novo servidor. Em VPS, use um tunel SSH para a porta 40120.

## Instalacao parcial experimental

Na interface, escolha Instalacao parcial experimental, informe um destino novo ou vazio, gere o plano, selecione um resource publico e confirme a instalacao parcial. O botao INSTALAR SERVIDOR baixa o release, compara o SHA-256 publicado com o catalogo e com o arquivo executado, inspeciona o ZIP e extrai para uma area temporaria privada antes de publicar no destino sem sobrescrever nada.

O diretorio gerado pela instalacao parcial NAO e um servidor pronto para iniciar: FXServer, dependencias externas e os demais resources nao sao instalados automaticamente. O `server.cfg` e os modelos de `cfg/` permanecem os oficiais; em `cfg/resources.cfg`, linhas de boot de resources ausentes ou com dependencia nao instalada ficam comentadas. Use apenas pastas descartaveis e nunca um servidor de producao. A correspondencia de hash nao comprova procedencia criptografica.

Camada de boot, configuracao e instalacao para subir um servidor FiveM usando `mz_core`.

## O que este repositorio e
- `mz_starter` organiza o `server.cfg`, os arquivos em `cfg/` e o fluxo basico de dependencias do ambiente.
- Ele existe para preparar e iniciar o servidor, nao para concentrar a logica do framework.

## O que este repositorio nao e
- `mz_starter` nao e o framework.
- A logica de dominio do servidor fica no `mz_core`: player, sessao, orgs, contas, inventario, veiculos e demais regras do core.

## Relacao com recursos oficiais
- O instalador sincroniza os recursos stock necessarios para o boot atual a partir do repositorio oficial `citizenfx/cfx-server-data`.
- Hoje ele traz `mapmanager`, `spawnmanager` e `baseevents`, mantendo o restante da base stock fora do escopo deste repositorio. `chat`, `yarn` e `webpack` sao fornecidos por `citizen/system_resources` nos artifacts atuais. O perfil suportado usa OneSync; portanto, nao instala nem inicia o `sessionmanager` legado de servidores sem OneSync.

## O que o instalador faz hoje
- baixa `oxmysql`
- baixa `ox_lib`
- sincroniza os recursos CFX base usados no `cfg/resources.cfg`
- clona ou atualiza `mz_core`
- clona ou atualiza `mz_notify`
- clona ou atualiza `mz_sync`
- clona ou atualiza os resources MZ listados no boot atual
- clona ou atualiza `screenshot-basic` em `resources/[standalone]`
- instala/reutiliza Bob74 e provisiona `mz_cayo` no grupo `resources/[maps]`
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
4. Se a distribuicao ainda contiver resources sem repositorio publico, aponte `MZ_RESOURCE_SOURCE_ROOT` para a raiz do pacote MZ completo.
5. Rode `tools/install_deps.bat` ou `tools/install_deps.sh`.
6. Rode `tools/check_structure.bat` (Windows) ou `bash tools/check_structure.sh` (Linux) e somente inicie o servidor com a estrutura completa.

No workspace oficial Windows, por exemplo:

```bat
set "MZ_RESOURCE_SOURCE_ROOT=D:\git-hub\repo_oficial"
tools\install_deps.bat
tools\check_structure.bat
```

`mz_warns` e `mz_computer` fazem parte do runtime atual. O instalador interrompe o clean install antes de alterar dependencias se eles - ou qualquer outro resource distribution-only - nao estiverem na raiz-fonte nem previamente instalados.

## Catalogo de opcoes do FXServer

Veja [cfg/fivem_options.example.cfg](cfg/fivem_options.example.cfg) para exemplos comentados de configuracao geral, OneSync, pure mode, rede, listagem, seguranca e integracoes opcionais.

O catalogo nao e carregado pelo `server.cfg`; todas as opcoes permanecem desativadas. Copie somente as linhas que decidir usar para os arquivos existentes em `cfg/`, sem duplicar nem substituir configuracoes ja definidas. Convars especificas de resources podem nao estar no catalogo e variam conforme o artifact/resource instalado.

## Documentacao
- [Instalacao](docs/install.md)
- [Estrutura do ambiente](docs/structure.md)

O boot declara `ensure bob74_ipl`, `ensure mz_settings` e `ensure mz_cayo`, nessa
ordem; nao substitua por `ensure [maps]`. O instalador migra um Bob antigo
somente quando o clone está limpo e validado, e falha sem sobrescrever cópias
alteradas ou duplicadas. Consulte [Instalacao](docs/install.md) para o
procedimento do layout antigo.

## screenshot-basic

O `screenshot-basic` e dependencia do `mz_feedback`, fica em `resources/[standalone]/screenshot-basic` e inicia antes dos resources MZ.

Ele sera usado em uma fase futura do `mz_realestate` para fotos de imoveis. No estado atual, anuncios, placas e detalhes publicos continuam funcionando sem implementar captura de imagens.

Instalacao manual, se necessario:

```sh
mkdir -p "resources/[standalone]"
git clone https://github.com/citizenfx/screenshot-basic.git "resources/[standalone]/screenshot-basic"
```

## pma-voice

O `pma-voice` e instalado em `resources/[som]/pma-voice` antes de `mz_phone` e `mz_radio`. O pin e o patch fazem parte do contrato de seguranca do telefone: apenas resources server-side podem atribuir membership de chamada. O instalador aborta sem sobrescrever quando encontra uma copia sem `.git` ou alteracoes locais diferentes do patch esperado.

Nao atualize esse resource com `git pull` isolado. Para atualizar, mude tag, commit e patch juntos nos dois instaladores, valide o spoof de `pma-voice:setPlayerCall` e execute as regressões de `mz_phone` e `mz_radio`.
