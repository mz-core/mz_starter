# MZ Starter 2.0: catalogo e planejamento

Esta etapa e somente leitura. Nao instala, atualiza, baixa assets, extrai ZIPs, altera configuracoes ou executa scripts do servidor. O catalogo `catalog/resources.json` e um retrato inicial; seus hashes e disponibilidades nao constituem autorizacao nem comprovam procedencia. Verifique metadados novamente antes de qualquer futura operacao de instalacao.

Execute os comandos a partir da raiz deste repositorio, com Go 1.22 ou posterior:

```sh
go run ./cmd/mz-starter catalog
go run ./cmd/mz-starter diagnose --server /caminho/do/server-data
go run ./cmd/mz-starter plan --server /caminho/do/server-data --workspace /caminho/do/workspace
go run ./cmd/mz-starter releases --resource mz_navigation --online
```

No Windows, substitua os caminhos pelos caminhos nativos, por exemplo `--server "D:\\FXServer\\server-data"`. `diagnose` e `plan` produzem o mesmo plano JSON nesta etapa. `--catalog` permite apontar outro catalogo validado. `--workspace` e opcional e compara os manifestos locais sem executa-los. Sem `--online`, nenhuma consulta de rede e feita. `plan --online` consulta metadados de releases dos resources; `releases --online` consulta um resource especifico. Um token opcional `GITHUB_TOKEN` e usado apenas na API oficial e nunca e incluido na saida.

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
