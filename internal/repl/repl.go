package repl

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/rendu-go/grimhold/internal/game"
	"github.com/rendu-go/grimhold/internal/open5e"
)

type cliCommand struct {
	name        string
	description string
	callback    func([]string) error
}

type REPL struct {
	game     *game.Game
	client   *open5e.Client
	commands map[string]cliCommand
	page     open5e.Page
	quit     bool
	savePath string
}

func New(g *game.Game, client *open5e.Client, savePath string) *REPL {
	r := &REPL{game: g, client: client, savePath: savePath}
	r.commands = map[string]cliCommand{
		"help":      {"help", "Afficher les commandes", r.help},
		"map":       {"map", "Afficher les régions et leurs sorties", r.worldMap},
		"travel":    {"travel <region>", "Rejoindre une région voisine", r.travel},
		"look":      {"look", "Observer la région", r.look},
		"explore":   {"explore", "Rencontrer un monstre", noArgs(g.Explore)},
		"fight":     {"fight", "Jouer un round de combat", noArgs(g.Fight)},
		"flee":      {"flee", "Tenter de fuir le combat", noArgs(g.Flee)},
		"character": {"character", "Afficher le personnage", r.character},
		"rest":      {"rest", "Se reposer une fois par visite", noArgs(g.Rest)},
		"bestiary":  {"bestiary", "Afficher la première page Open5e", r.bestiary},
		"next":      {"next", "Afficher la page suivante", r.next},
		"prev":      {"prev", "Afficher la page précédente", r.prev},
		"inspect":   {"inspect <monster>", "Afficher les statistiques d'un monstre", r.inspect},
		"slain":     {"slain", "Afficher les monstres vaincus et les compteurs", r.slain},
		"save":      {"save", "Sauvegarder dans save.json", r.save},
		"load":      {"load", "Charger save.json", r.load},
		"quit":      {"quit", "Quitter le jeu", r.exit},
	}
	return r
}

func (r *REPL) Run(input io.Reader) error {
	fmt.Println("Grimhold. Tapez help pour afficher les commandes.")
	r.game.Look()
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for !r.quit {
		fmt.Print("grimhold > ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("lecture de la commande : %w", err)
			}
			fmt.Println()
			return nil
		}
		if err := r.Execute(scanner.Text()); err != nil {
			fmt.Println("Erreur :", err)
		}
	}
	return nil
}

func (r *REPL) Execute(line string) error {
	parts := strings.Fields(strings.ToLower(line))
	if len(parts) == 0 {
		return nil
	}
	command, found := r.commands[parts[0]]
	if !found {
		return fmt.Errorf("commande inconnue : %s. Tapez help", parts[0])
	}
	return command.callback(parts[1:])
}

func noArgs(callback func() error) func([]string) error {
	return func(args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("cette commande ne prend aucun argument")
		}
		return callback()
	}
}

func checkArgs(args []string, count int, usage string) error {
	if len(args) != count {
		return fmt.Errorf("utilisation : %s", usage)
	}
	return nil
}

func (r *REPL) help(args []string) error {
	if err := checkArgs(args, 0, "help"); err != nil {
		return err
	}
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		command := r.commands[name]
		fmt.Printf("  %-20s %s\n", command.name, command.description)
	}
	return nil
}

func (r *REPL) worldMap(args []string) error {
	if err := checkArgs(args, 0, "map"); err != nil {
		return err
	}
	for _, region := range r.game.World.Regions {
		fmt.Printf("%s (%s), danger %d/5, sorties : %s\n", region.ID, region.Name, region.Danger, strings.Join(region.Exits, ", "))
	}
	return nil
}

func (r *REPL) travel(args []string) error {
	if err := checkArgs(args, 1, "travel <region>"); err != nil {
		return err
	}
	if err := r.game.Travel(args[0]); err != nil {
		return err
	}
	r.game.Look()
	return nil
}

func (r *REPL) look(args []string) error {
	if err := checkArgs(args, 0, "look"); err != nil {
		return err
	}
	r.game.Look()
	return nil
}

func (r *REPL) character(args []string) error {
	if err := checkArgs(args, 0, "character"); err != nil {
		return err
	}
	r.game.CharacterSheet()
	return nil
}

func (r *REPL) showPage(address string) error {
	page, err := r.client.List(address)
	if err != nil {
		return err
	}
	r.page = page
	for _, monster := range page.Results {
		fmt.Printf("%s : %s, danger %s\n", monster.Slug, monster.Name, monster.ChallengeRating)
	}
	fmt.Printf("Page précédente : %t. Page suivante : %t.\n", page.Previous != "", page.Next != "")
	return nil
}

func (r *REPL) bestiary(args []string) error {
	if err := checkArgs(args, 0, "bestiary"); err != nil {
		return err
	}
	return r.showPage("")
}

func (r *REPL) next(args []string) error {
	if err := checkArgs(args, 0, "next"); err != nil {
		return err
	}
	if r.page.Next == "" {
		return fmt.Errorf("aucune page suivante. Utilisez bestiary pour commencer")
	}
	return r.showPage(r.page.Next)
}

func (r *REPL) prev(args []string) error {
	if err := checkArgs(args, 0, "prev"); err != nil {
		return err
	}
	if r.page.Previous == "" {
		return fmt.Errorf("aucune page précédente")
	}
	return r.showPage(r.page.Previous)
}

func (r *REPL) inspect(args []string) error {
	if err := checkArgs(args, 1, "inspect <monster>"); err != nil {
		return err
	}
	monster, err := r.client.Monster(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("%s (%s)\nPV : %d\nArmure : %d\nDanger : %s\n", monster.Name, monster.Slug, monster.HitPoints, monster.ArmorClass, monster.ChallengeRating)
	for _, action := range monster.Actions {
		fmt.Printf("%s : %s\n", action.Name, action.Description)
		if action.DamageDice != "" {
			fmt.Printf("Attaque : %+d, dégâts : %s %+d\n", action.AttackBonus, action.DamageDice, action.DamageBonus)
		}
	}
	return nil
}

func (r *REPL) slain(args []string) error {
	if err := checkArgs(args, 0, "slain"); err != nil {
		return err
	}
	if len(r.game.State.Slain) == 0 {
		fmt.Println("Vous n'avez encore vaincu aucun monstre.")
		return nil
	}
	names := make([]string, 0, len(r.game.State.Slain))
	for name := range r.game.State.Slain {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s : %d\n", name, r.game.State.Slain[name])
	}
	return nil
}

func (r *REPL) save(args []string) error {
	if err := checkArgs(args, 0, "save"); err != nil {
		return err
	}
	if err := r.game.Save(r.savePath); err != nil {
		return err
	}
	fmt.Println("Partie sauvegardée.")
	return nil
}

func (r *REPL) load(args []string) error {
	if err := checkArgs(args, 0, "load"); err != nil {
		return err
	}
	if err := r.game.Load(r.savePath); err != nil {
		return err
	}
	fmt.Println("Partie chargée.")
	r.game.Look()
	return nil
}

func (r *REPL) exit(args []string) error {
	if err := checkArgs(args, 0, "quit"); err != nil {
		return err
	}
	r.quit = true
	fmt.Println("À bientôt.")
	return nil
}
