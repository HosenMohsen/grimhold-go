package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rendu-go/grimhold/internal/game"
	"github.com/rendu-go/grimhold/internal/open5e"
	"github.com/rendu-go/grimhold/internal/repl"
	"github.com/rendu-go/grimhold/internal/world"
)

func run() error {
	resume := flag.Bool("continue", false, "Charger save.json au démarrage")
	flag.Parse()
	w, err := world.Load("world.json")
	if err != nil {
		return err
	}
	client := open5e.NewClient(open5e.BaseURL, 5*time.Minute)
	defer client.Close()
	g := game.New(w, client)
	if *resume {
		if err := g.Load("save.json"); err != nil {
			fmt.Println("Erreur :", err)
		}
	}
	return repl.New(g, client, "save.json").Run(os.Stdin)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur :", err)
		os.Exit(1)
	}
}
