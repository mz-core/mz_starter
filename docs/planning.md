# MZ Starter 2.0: catalogo e planejamento

O planejamento nao altera o servidor. A instalacao parcial experimental, apos confirmacao explicita, escreve em um destino novo ou vazio; nao atualiza nem executa scripts. A verificacao opcional de pacotes baixa assets para uma area temporaria privada e os remove apos inspecao. O catalogo `catalog/resources.json` e um retrato inicial; seus hashes e disponibilidades nao constituem autorizacao nem comprovam procedencia. Verifique metadados novamente antes de qualquer futura operacao de instalacao.

Os executaveis portateis usam por padrao o catalogo incorporado, inclusive fora do repositorio. Para desenvolvimento com Go 1.22 ou posterior, na raiz deste repositorio:

```sh
go run ./cmd/mz-starter catalog
go run ./cmd/mz-starter diagnose --server /caminho/do/server-data
go run ./cmd/mz-starter plan --server /caminho/do/server-data --workspace /caminho/do/workspace
go run ./cmd/mz-starter releases --resource mz_navigation --online
```

No Windows, substitua os caminhos pelos caminhos nativos, por exemplo `--server "D:\\FXServer\\server-data"`. `diagnose` e `plan` produzem o mesmo plano JSON nesta etapa. `--catalog` permite apontar outro catalogo, que e lido e validado; caminho vazio, arquivo inexistente ou catalogo invalido causam erro. `--workspace` e opcional e compara os manifestos locais sem executa-los. Sem `--online`, nenhuma consulta de rede e feita. `plan --online` consulta metadados de releases dos resources; `releases --online` consulta um resource especifico. Um token opcional `GITHUB_TOKEN` e usado apenas na API oficial e nunca e incluido na saida.

O JSON de plano tem `schema_version`, `mode`, `server`, `steps`, `issues` e `ready`. Cada passo fornece ID, acao simulada, estado, versoes observadas, procedencia e bloqueios. `ready` permanece `false` enquanto nao existir verificacao de procedencia e compatibilidade confiavel. `integrity_unverified` significa que a API informou um digest bem formado, nao que o ZIP tenha sido baixado ou verificado. Uma marca `.mz-starter.json` coerente e relatada como `managed_record_unverified`: ela nao comprova integridade nem autoriza sobrescrita.

O provider consulta a lista de releases, nao somente `latest`. Uma tag `vX.Y.Z` ou `<resource>-vX.Y.Z` e reconhecida; tags prefixadas recebem `legacy_tag`. Um ZIP ausente, nome inesperado, multiplos ZIPs, digest ausente/invalido, rate limit e falhas de acesso sao estados distintos. Uma resposta 404 e `not_found_or_no_access`, pois o GitHub pode ocultar repositorios privados. Releases retornados sao observacoes; nenhum asset e selecionado para instalacao.

Quando existem releases, mas nenhuma tag e reconhecida como SemVer, o estado e `no_recognized_releases` e as tags aparecem em `unrecognized_tags`; isso difere de `release_missing`.

Para um ID MZ ainda sem repositorio no catalogo, a consulta online testa apenas o candidato `mz-core/<id>`; isso nao declara sua existencia nem altera a fonte autorizada. O bloqueio de distribuicao so pode ser resolvido por atualizacao e revisao do catalogo. Apos rate limit, o plano nao continua enviando chamadas ao GitHub.

O parser de manifesto e estatico e limitado a versoes e dependencias literais. Expressoes Lua dinamicas nao sao executadas nem comprovadas. O perfil segue a ordem atual do boot e relata conflitos em vez de corrigi-los. Artefatos e dependencias externas permanecem `external_unverified`. O banguard usa somente a fonte `protected`; nao existe fallback para codigo-fonte privado.

Os testes Go ficam em `mz_testkit/tests/mz_starter` e podem ser executados de la com `go test ./...`. A validacao automatizada nao representa teste FiveM ou aprovacao para instalacao real.

## Interface local (Wave 2)

A interface incorpora o frontend e o mesmo catalogo no executavel Go. `serve` aceita apenas conexoes em `127.0.0.1`, usa porta efemera por padrao e imprime um endereco com uma credencial aleatoria de sessao no fragmento. Abra o endereco completo no navegador; nao compartilhe esse endereco. A API valida Host, Origin e token, e so oferece leitura e planejamento. O processo precisa permanecer aberto. O token `GITHUB_TOKEN`, quando fornecido, e usado somente no servidor para consultar a API oficial; nunca o insira na interface.

No Windows, inicie `mz-starter.exe serve`. No Linux grafico, execute `./mz-starter serve`; o programa tenta abrir o navegador e tambem imprime o endereco. Para selecionar porta e nao abrir navegador, use `serve --port 48123 --no-open`. Em VPS headless, mantenha esse processo em execucao e, em outro terminal **local**, crie o tunel `ssh -L 48123:127.0.0.1:48123 usuario@servidor`. Abra no computador local o endereco completo impresso pelo Starter, com a mesma porta e fragmento. O servico continua vinculado ao loopback da VPS; nao o exponha publicamente. A permissao de leitura do processo limita os arquivos acessiveis, e usuarios capazes de ler seu terminal ou controlar o mesmo usuario local devem ser considerados confiaveis.

No assistente, informe o diretorio `server-data` que contem `cfg/resources.cfg`; o workspace opcional permite comparar manifestos fonte. A verificacao online e opcional, pode demorar e consulta somente metadados. Um 404 do GitHub continua ambiguo entre ausencia e falta de acesso. Os hashes exibidos nao foram calculados dos arquivos. Os dados de disponibilidade do catalogo sao um retrato anterior. Instalacao, atualizacao, reparacao e rollback nao estao implementados.

### Build reproduzivel

Node.js e npm sao necessarios apenas na maquina de desenvolvimento para compilar o frontend; Go 1.22 ou posterior compila o executavel. Na raiz de `mz_starter`:

```sh
cd web
npm ci
npm run build
cd ..
go build -o mz-starter ./cmd/mz-starter
```

Para o Windows, use `go build -o mz-starter.exe ./cmd/mz-starter`. Para compilacao cruzada, defina `GOOS=windows` ou `GOOS=linux` e `GOARCH=amd64` antes do build. O frontend em `web/dist` deve ser gerado antes do Go, pois seus arquivos e o catalogo sao incorporados via `go:embed`. O pacote final nao precisa de Node.js nem de servidor web separado.

Se o Go nao estiver no PATH, use o caminho absoluto do executavel Go em cada comando, por exemplo `& "C:\caminho\go\bin\go.exe" build -o mz-starter.exe ./cmd/mz-starter` no PowerShell ou `/caminho/go/bin/go build -o mz-starter ./cmd/mz-starter` no Linux. Nao e necessario alterar PATH nem instalar globalmente. Os testes continuam em `mz_testkit/tests/mz_starter`: execute `go test ./...` a partir desse diretorio.

### Homologacao portatil

A interface informa erro quando `cfg/resources.cfg` nao pode ser lido ou nao contem recursos; uma tentativa falha descarta o plano anterior da sessao. Para a homologacao, extraia cada pacote em pasta nova e use apenas um `server-data` descartavel. Nao sao enviados dados de configuracao ou arquivos de servidor ao GitHub; a opcao online consulta metadados das fontes do catalogo.

Os testes portateis estao em `mz_testkit/tests/mz_starter/portable_smoke.py` e `portable_linux.sh`. Eles operam com fixtures descartaveis e verificam conteudo do ZIP, catalogo e plano em loopback; o teste Linux deve ser executado como usuario sem privilegios. Uma URL impressa contem um token temporario de sessao: nao copie essa URL para logs compartilhados.

### Teste a partir do GitHub Releases

Baixe o ZIP da sua plataforma e `SHA256SUMS.txt` em https://github.com/mz-core/mz_starter/releases. Confirme o SHA-256 antes de extrair em uma pasta nova. Os arquivos `LEIA-ME.txt` de cada ZIP descrevem a execucao e o teste com fixture descartavel sem o workspace. No Windows, use `Get-FileHash -Algorithm SHA256 <zip>`. No Linux, use `sha256sum -c SHA256SUMS.txt --ignore-missing`, depois `chmod u+x ./mz-starter-linux-amd64` e `./mz-starter-linux-amd64 serve --port 48123 --no-open`. Para acesso remoto, conecte `ssh -L 48123:127.0.0.1:48123 usuario@vps` e abra a URL completa de sessao no navegador local. Nunca use a instalacao FiveM de producao como fixture de homologacao.

## Planejamento de bootstrap (Wave 2.6)

No assistente, selecione explicitamente **Nova instalacao** ou **Servidor existente**. Nova instalacao aceita uma pasta vazia ou ainda inexistente e apresenta os dez arquivos de configuracao/icone planejados, configuracoes necessarias, recursos e seis fases futuras. Nenhuma pasta ou arquivo e criado. O patch oficial de pma-voice fica incorporado para revisao futura; nao e aplicado. O unico perfil suportado e o perfil observado no catalogo; nao existe selecao parcial de recursos nesta etapa.

O modo existente exige `server.cfg` como arquivo regular e `cfg/resources.cfg` legivel com recursos e apenas os analisa. Uma estrutura parcial e informada como ambigua; uma pasta nao vazia nao e aceita como instalacao nova. Configuracoes existentes jamais sao sobrescritas. As fontes dos modelos sao os arquivos oficiais versionados de `mz_starter`, incorporados no build, com verificacao dos placeholders sensiveis e da ordem do boot. `cfg/fivem_options.example.cfg` e apenas referencia opcional, nao e carregado nem planejado como arquivo necessario.

Na CLI, mantendo a compatibilidade dos comandos anteriores, use:

```sh
go run ./cmd/mz-starter plan --installation-mode new --server /caminho/pasta-vazia
go run ./cmd/mz-starter plan --installation-mode existing --server /caminho/server-data
```

O plano inclui `installation_mode`, `profile`, `bootstrap_files`, `required_configuration` e `phases`. Sao contratos descritivos, nao instrucoes executaveis. Licenca, conexao de banco, endpoints, identificadores de owner e URLs/segredo de telefone exigem configuracao futura; nenhum valor real e solicitado ou gravado agora. O resultado nunca autoriza instalacao sem verificacao de releases, integridade, autorizacao e comportamento transacional.

### Abertura direta

No Windows, execute `mz-starter.exe` sem argumentos para iniciar a interface e abrir o navegador. O terminal apresenta tambem a URL local da sessao caso o navegador nao abra; mantenha-o aberto e encerre com Ctrl+C. A URL contem um token temporario e nao deve ser compartilhada. `mz-starter.exe serve` continua disponivel. Em Linux headless, use `./mz-starter-linux-amd64 serve --port 48123 --no-open` e um tunel SSH. Use `--help` para consultar os comandos.


## Verificacao de pacotes (Wave 3A)

Em Recursos, abra os detalhes, escolha opcionalmente uma tag reconhecida e selecione Verificar pacote. O backend resolve somente repositorios autorizados do catalogo e assets de release; nao aceita URL do navegador nem usa arquivos Source Code. A autenticacao de sessao, Host e Origin aplica-se tambem aos endpoints de verificacao. A consulta e o download utilizam o token opcional `GITHUB_TOKEN` somente na API GitHub; redirecionamentos para servidores de assets autorizados nao recebem o token. A ausencia de autenticacao para fonte privada bloqueia a operacao; `mz_banguard` permanece protegido.

O limite padrao e 512 MiB por download, configuravel com `serve --max-download-mib` (1 a 2048). A inspecao limita a 10000 entradas, 1 GiB descomprimido, 256 MiB por arquivo e razao 200 para arquivos acima de 1 MiB. As operacoes sao limitadas a duas simultaneas e podem ser canceladas. O ZIP temporario e removido apos a inspecao; falhas de limpeza sao relatadas. A area temporaria nao pode estar dentro de um diretorio de servidor informado na sessao. Nenhum arquivo e extraido nem executado.

A igualdade SHA-256 comprova somente a correspondencia com o digest da API. A verificacao estrutural cobre caminhos, nomes reservados, colisoes por caixa, links, limites, CRC, manifesto literal e arquivos obrigatorios do catalogo. Nao valida automaticamente todos os padroes glob ou arquivos referenciados dinamicamente e nao comprova procedencia nem que o pacote corresponde ao commit. A instalacao continua bloqueada. O painel apresenta os dados reais e estados separados para digest, estrutura e procedencia.

## Hardening e contrato de procedencia (Wave 3A.1)

O verificador analisa estaticamente `client_script(s)`, `server_script(s)`, `shared_script(s)`, `file(s)`, `ui_page` e `data_file`, incluindo `*` e `**` suportados. Caminhos literais ausentes, globs sem correspondencia, expressoes dinamicas e referencias externas sao reportados; `LoadResourceFile` local com caminho literal tambem e verificado; se o arquivo nao existir, sua obrigatoriedade permanece inconclusiva porque a condicao de execucao nao e provada. Declaracoes nao compreendidas permanecem inconclusivas. Quando o pacote contem scripts Lua ou JavaScript alem do manifesto, a cobertura de referencias de runtime permanece explicitamente inconclusiva; a verificacao estatica nao declara o pacote completo apenas porque nao encontrou uma chamada dinamica. Nao e uma execucao Lua: o carregamento dinamico arbitrario em Lua ou JavaScript, referencias web e cobertura de todos os tiles nao podem ser provados apenas pelo manifesto. `structure_incomplete` impede tratar essas verificacoes como comprovacao de completude.

O diretorio temporario Linux e exclusivo por operacao (0700); a raiz deve ser do usuario ou root e, se gravavel por terceiros, ter sticky bit. No Windows, exige-se volume local fixo com ACLs persistentes; a raiz padrao e LocalAppData, e nao TEMP: o diretorio e criado com DACL protegida para o SID do usuario atual, conferida apos a criacao. Ancestrais sao recusados se tiverem proprietario nao confiavel, reparse point, ACL incompreensivel ou ACE de escrita/remocao que permita substituicao por terceiro; somente o usuario, SYSTEM, Administradores e TrustedInstaller sao tratados como confiaveis. Se a verificacao falhar, a operacao para com `temporary_security_unavailable` antes de gravar o ZIP. No ambiente auditado o TEMP concedia modificacao a outro SID e foi recusado. Os testes nao abrangem toda configuracao de Windows, ACLs de rede ou processos maliciosos com a mesma identidade/privilegios administrativos.

### Contrato proposto de evidencia

Uma evidencia futura deve vincular, num documento versionado e assinado, identificador do repositorio oficial, ref/tag e commit resolvido, versao do manifesto, identidade e versao do workflow/empacotador, entradas e lockfiles/ferramentas, nome e SHA-256 do ZIP final. A verificacao deve validar assinatura e cadeia de confianca, identidade do emissor/workflow e repositorio, ref/commit imutavel e digest do ZIP baixado; tudo que faltar resulta em `provenance_unverified`. Um manifesto nao assinado ou o digest da propria API GitHub nao estabelecem essa cadeia.

O `tools/release/release.py` ja confere tag contra `GITHUB_SHA` quando presente, coleta arquivos e gera ZIP com entradas ordenadas e timestamps fixos; o passo `verify` compara com referencia e blobs. Futuramente o workflow pode gerar uma atestacao sobre o SHA-256 do ZIP e publicar a evidencia junto ao release, vinculando o commit efetivo e o workflow confiavel. Nao existe hoje atestacao validada pelo Starter; nao se promove nenhum pacote a procedencia comprovada automaticamente. A reproducibilidade tambem exige controle de toolchains, dependencias e insumos, alem dos timestamps do ZIP.

Referencias: https://docs.fivem.net/docs/scripting-reference/resource-manifest/ e https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations . A verificacao de atestacoes exige validar criptografia e identidade do emissor, nao apenas obter o arquivo: https://docs.github.com/en/rest/orgs/attestations .

Validacoes ja possiveis: identidade/restricao da URL no catalogo; consulta da tag e asset pela API; comparacao de versao declarada com o manifesto literal; digest do asset recebido; e resolucao da ref Git para um commit pela API GitHub (incluindo dereferencia de tag anotada). A implementacao atual ainda nao resolve nem autentica o commit, portanto sua procedencia permanece nao comprovada. Para um contrato verificavel, alem da assinatura, o consumidor deve exigir o repositorio e workflow aprovados, digest, commit e ref pretendidos, e tratar ausencia, identidade divergente, revogacao ou falha de verificacao como inconclusivos/bloqueantes. A associacao de uma tag no GitHub, isoladamente, nao prova que o workflow compilou esse commit nem que o ZIP e seu resultado.

Em uma verificacao real do asset oficial `mz_navigation-v1.1.0.zip`, o SHA-256 calculado coincidiu com o digest informado pelo GitHub: `227dc45c724b0a48929553dc0b892cec225c6a22211dc03296271c2e10912080`. O ZIP contem `web/assets/los-santos-atlas.webp` e `web/assets/atlas.json`; o codigo distribuido usa o atlas estatico e desabilita tiles por padrao. A leitura de `web/tiles/atlas.json` esta condicionada a habilitacao dos tiles. O arquivo e os tiles JPG/PNG estao ausentes; uma leitura literal nao prova que seja obrigatoria. O verificador generico relata a ausencia e os padroes sem correspondencia como inconclusivos, sem classifica-los automaticamente como opcionais. Referencias externas e dinamicas tambem permanecem inconclusivas. A coincidencia do digest nao comprovou a origem do build ou a correspondencia com o commit.

### Gates e proposta de atestacao (Wave 3A.2)

Beta.4 experimental: a instalacao parcial real esta habilitada apos confirmacao explicita, com staging privado, verificacao de digest, rename sem sobrescrita e boot parcial explicitamente nao pronto. O servidor gerado nao deve ser iniciado. Antes de qualquer instalacao completa, homologar em Windows e Linux/VPS o isolamento entre usuarios, falhas de limpeza, ACL, cancelamento durante download/extracao, concorrencia e limitacoes de rename na mesma unidade. Wave 3B: NO-GO ate procedencia verificada, politica de referencias, instalacao transacional completa, backup, rollback e homologacao real.

Proposta para aprovacao: no GitHub Actions do repositorio oficial, resolver a tag para commit imutavel, conferir o checkout e gerar o ZIP pelo empacotador existente. Com permissoes minimas `contents: read`, `id-token: write` e `attestations: write`, usar `actions/attest-build-provenance` sobre o ZIP final e publicar exatamente esse asset. O consumidor deve verificar assinatura e cadeia confiavel da atestacao, digest do ZIP, identidade do repositorio e workflow autorizado e commit esperado para a tag. Valores declarados dentro do ZIP e o digest da API, isoladamente, nao comprovam origem. O Starter ainda nao implementa essa validacao.

### Contrato de arquivos para homologacao

O catalogo incorpora `file_rules` por resource, com versao, SHA-256 exato do ZIP, caminho ou padrao, classificacao, condicao, evidencia e origem. Regras sao dados de curadoria do Starter, nao entradas confiadas ao ZIP. Apenas o mesmo digest e versao ativam o registro; divergencia deixa o perfil inconclusivo. Arquivos `required` ausentes nao sao neutralizados por regras condicionais. Integridade, referencias, perfil e procedencia sao informados separadamente; instalacao completa permanece indisponivel e a instalacao parcial experimental exige as condicoes descritas adiante.

No `mz_navigation` 1.1.0, a revisao registra os dois arquivos de atlas estatico como requeridos e os caminhos de tiles como condicionais. O digest fixado corresponde ao pacote cuja configuracao distribuida desabilita tiles. Esta evidencia descreve apenas aquele pacote: nao analisa mudancas posteriores, nao comprova configuracao em runtime e nao calcula automaticamente se uma condicao foi satisfeita. Com tiles habilitados e ausentes, a funcionalidade nao deve ser aprovada; o codigo do resource informa fallback quando o atlas de tiles esta ausente. Uma configuracao dinamica ou alterada e inconclusiva. A cobertura de referencias continua inconclusiva onde ha runtime dinamico.

O contrato atual registra classificacoes mas ainda nao calcula a compatibilidade automatica de perfis habilitados ou alterados. Nao usar esse resultado para instalar; concluir esses cenarios e sua homologacao antes de considerar esta etapa fechada.

### Preparacao de um build rastreavel

Para um candidato de publicacao futuro, use uma revisao ja aprovada, com `git status --porcelain` vazio, registre `git rev-parse HEAD` e `git rev-parse HEAD^{tree}`, instale as dependencias fixadas em `web/package-lock.json` com `npm ci`, execute `npm run build` e confira novamente `git status --porcelain`. Se o frontend gerado diferir da arvore aprovada, nao associe o executavel ao commit limpo: revise e versione a alteracao antes de reconstruir. Registre versoes de Go, Node e npm, sistema, variaveis `GOOS`, `GOARCH`, `CGO_ENABLED`, arquivos e SHA-256 finais. So associe o build ao commit quando todas as entradas utilizadas estiverem comprovadamente nessa arvore. Os pacotes locais produzidos com alteracoes nao commitadas servem somente para homologacao e nao comprovam reproducibilidade a partir do HEAD.

## Instalacao completa do MZ Core

O assistente oferece Instalacao completa do MZ Core. A resolucao consulta os releases reais dos repositorios oficiais e monta os 51 componentes do perfil antes de qualquer escrita. Resources publicos usam a API do GitHub para identificar tag, asset e digest e baixam pela URL publica do release; se a API estiver temporariamente limitada, o catalogo fixado e usado como fallback e o arquivo continua verificado pelo SHA-256. `mz_police` e publico e passou a ser resolvido automaticamente. `mz_core`, `mz_inventory` e `mz_admin` foram revalidados contra o digest atual; `mz_vehicles` continua sem asset no release atual e exige pacote local.

No perfil sem distribuicao local nem token, 26 releases MZ, 7 snapshots externos, `baseevents` e 2 system resources resolvem; 16 resources ficam bloqueados. Desses, 3 (`mz_prison`, `mz_furniture`, `mz_admin`) possuem releases privados e podem ser baixados com token autorizado; os outros 13 nao possuem release publica utilizavel. O operador pode escolher instalar somente os componentes disponiveis; o resultado e `incomplete` e nunca e declarado completo.

A distribuicao local pode ser uma pasta com pacotes `<id>-v<versao>.zip`/`<id>.zip` ou com as pastas dos resources. Ela exige autorizacao explicita, exclui `.git`, `node_modules`, `graphify-out`, `tests`, `test-results`, logs, caches, `.env` e fontes web de desenvolvimento, rejeita links e calcula um digest canonico da arvore copiada. `mz_banguard` nunca e consultado no GitHub e so pode vir dessa fonte local autorizada.

As dependencias externas usam contratos fixados e verificados: `oxmysql` 2.14.3, `ox_lib` 3.42.2, `pma-voice` 7.0.0 no commit aprovado com o patch server-authoritative, `bob74_ipl` 2.7.1, `screenshot-basic` no commit fixado e o snapshot fixado de `cfx-server-data` para `mapmanager`, `spawnmanager` e `baseevents`. Cada snapshot possui SHA-256 esperado; divergencia falha fechada.

Os artifacts do FXServer ficam separados do server-data. O modo `existing` valida `FXServer.exe`/`run.sh`; o modo `auto` baixa o build oficial recomendado no Windows e no Linux (ZIP ou `fx.tar.xz`) e o instala em `mz-artifacts` ao lado do server-data; o modo `download` permite escolher o destino. Os system resources fornecidos pelo artifact habilitam `yarn`, `webpack`, `screenshot-basic` e `mz_feedback`. O executor faz staging unico, download com concorrencia de metadados limitada, verificacao por componente, progresso, cancelamento e publicacao atomica sem sobrescrita. Um bloqueio impede a publicacao total; uma instalacao explicitamente incompleta publica apenas os componentes disponiveis e informa os ausentes. Licenca, banco, endpoints e permissoes continuam pendentes e o servidor nao e declarado pronto para iniciar.

## Instalacao parcial experimental

Na interface, escolha Nova instalacao, informe um destino novo ou vazio, gere o plano, escolha um resource publico e confirme explicitamente a instalacao parcial. O botao INSTALAR SERVIDOR baixa o release pelo verificador, exige que o digest do asset publicado, o digest do catalogo e o SHA-256 dos bytes realmente executados sejam o mesmo valor e extrai a partir do mesmo descritor de arquivo ja verificado para staging privado no mesmo diretorio pai. Arquivos de bootstrap sao os modelos incorporados e `server.cfg` permanece byte a byte igual ao modelo oficial; `cfg/resources.cfg` recebe um aviso explicito de que a instalacao parcial nao esta pronta para iniciar, e linhas de boot de resources nao instalados ou cujas dependencias diretas ou transitivas nao foram instaladas sao comentadas. Ciclos tambem impedem ativacao. A publicacao usa rename sem sobrescrita; um destino que ficou nao vazio e recusado. A operacao relata download, extracao, publicacao, falha e cancelamento. O cancelamento deixa de ser aceito quando comeca a publicacao atomica.

O modo parcial nao instala FXServer nem dependencias externas automaticamente e nunca afirma que o servidor esta pronto para iniciar. `mz_navigation` 1.1.0 tem regras revisadas e vinculadas a um digest especifico para o atlas padrao; referencias dinamicas continuam inconclusivas. A verificacao de hash nao comprova procedencia. Recursos privados, protegidos, com divergencias ou sem releases autorizados permanecem bloqueados. O diretorio pai deve permitir criar staging seguro e a mesma unidade precisa suportar rename atomico sem sobrescrita. Configure licenca, banco, endpoints e dependencias antes de usar um servidor futuro.
