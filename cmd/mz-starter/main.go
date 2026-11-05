package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mz-core/mz_starter/engine"
)

func main() {
	if len(os.Args) < 2 {
		fail("uso: mz-starter catalog|diagnose|plan|releases [opcoes]")
	}
	command := os.Args[1]
	if command == "serve" {
		serve(os.Args[2:])
		return
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	catalogPath := flags.String("catalog", "catalog/resources.json", "caminho do catalogo")
	server := flags.String("server", ".", "raiz do server-data")
	workspace := flags.String("workspace", "", "raiz opcional de manifestos fonte")
	resource := flags.String("resource", "", "ID para consulta de releases")
	online := flags.Bool("online", false, "consultar metadados GitHub")
	if err := flags.Parse(os.Args[2:]); err != nil {
		fail(err.Error())
	}
	if flags.NArg() != 0 {
		fail("argumentos adicionais nao suportados")
	}
	c, err := engine.LoadCatalog(*catalogPath)
	if err != nil {
		fail(err.Error())
	}
	out := json.NewEncoder(os.Stdout)
	out.SetIndent("", "  ")
	switch command {
	case "catalog":
		err = out.Encode(c)
	case "releases":
		var found *engine.Resource
		for i := range c.Resources {
			if c.Resources[i].ID == *resource {
				found = &c.Resources[i]
				break
			}
		}
		if found == nil {
			fail("resource nao encontrado no catalogo")
		}
		if !*online {
			err = out.Encode(engine.QueryResult{Status: "offline", Releases: []engine.Release{}})
			break
		}
		repository := found.Repository
		if repository == "" && found.Type == "fivem_resource" {
			repository = "https://github.com/mz-core/" + found.ID
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		g := engine.GitHub{Token: os.Getenv("GITHUB_TOKEN")}
		err = out.Encode(g.Releases(ctx, found.ID, repository))
	case "diagnose", "plan":
		boot, e := engine.ReadServerBoot(*server)
		if e != nil {
			fail("nao foi possivel ler cfg/resources.cfg: " + e.Error())
		}
		remote := map[string]engine.QueryResult{}
		if *online {
			g := engine.GitHub{Token: os.Getenv("GITHUB_TOKEN")}
			limited := false
			for _, r := range c.Resources {
				if r.Source == "protected" || r.Type != "fivem_resource" {
					continue
				}
				repository := r.Repository
				if repository == "" {
					repository = "https://github.com/mz-core/" + r.ID
				}
				if limited {
					remote[r.ID] = engine.QueryResult{Status: "not_queried_rate_limit", Releases: []engine.Release{}}
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				remote[r.ID] = g.Releases(ctx, r.ID, repository)
				limited = remote[r.ID].Status == "rate_limited"
				cancel()
			}
		}
		err = out.Encode(engine.BuildPlan(c, boot, *server, *workspace, remote))
	default:
		fail("comando desconhecido")
	}
	if err != nil {
		fail(err.Error())
	}
}
func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(2) }
