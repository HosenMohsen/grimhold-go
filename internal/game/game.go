package game

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/rendu-go/grimhold/internal/open5e"
	"github.com/rendu-go/grimhold/internal/world"
)

type MonsterSource interface {
	Monster(string) (open5e.Monster, error)
}

type Weapon struct {
	Name  string `json:"name"`
	Dice  string `json:"dice"`
	Bonus int    `json:"bonus"`
}

type Character struct {
	HP          int    `json:"hp"`
	MaxHP       int    `json:"max_hp"`
	Level       int    `json:"level"`
	XP          int    `json:"xp"`
	AttackBonus int    `json:"attack_bonus"`
	ArmorClass  int    `json:"armor_class"`
	Weapon      Weapon `json:"weapon"`
}

type Encounter struct {
	Monster open5e.Monster `json:"monster"`
	HP      int            `json:"hp"`
}

type State struct {
	Location  string         `json:"location"`
	Character Character      `json:"character"`
	Slain     map[string]int `json:"slain"`
	Gold      int            `json:"gold"`
	Rested    bool           `json:"rested"`
	Encounter *Encounter     `json:"encounter,omitempty"`
}

type Game struct {
	World  world.World
	State  State
	source MonsterSource
}

func New(w world.World, source MonsterSource) *Game {
	return &Game{
		World:  w,
		source: source,
		State: State{
			Location: w.Start,
			Slain:    make(map[string]int),
			Character: Character{
				HP: 30, MaxHP: 30, Level: 1, AttackBonus: 5, ArmorClass: 14,
				Weapon: Weapon{Name: "Épée", Dice: "1d8", Bonus: 3},
			},
		},
	}
}

func (g *Game) Travel(id string) error {
	if g.State.Encounter != nil {
		return fmt.Errorf("terminez le combat ou utilisez flee")
	}
	region, _ := g.World.Find(g.State.Location)
	for _, exit := range region.Exits {
		if exit == id {
			g.State.Location = id
			g.State.Rested = false
			return nil
		}
	}
	return fmt.Errorf("aucun chemin vers %s depuis %s", id, region.ID)
}

func (g *Game) Look() {
	r, _ := g.World.Find(g.State.Location)
	fmt.Printf("%s (%s)\n%s\nDanger : %d/5\nSorties : %s\n", r.Name, r.ID, r.Description, r.Danger, strings.Join(r.Exits, ", "))
	if g.State.Encounter != nil {
		fmt.Printf("Face à vous : %s, %d PV. Utilisez fight ou flee.\n", g.State.Encounter.Monster.Name, g.State.Encounter.HP)
	}
}

func (g *Game) Explore() error {
	if g.State.Encounter != nil {
		return fmt.Errorf("un monstre vous barre déjà la route")
	}
	r, _ := g.World.Find(g.State.Location)
	var foe open5e.Monster
	for i := 0; i < r.Danger; i++ {
		slug := r.Foes[rand.IntN(len(r.Foes))]
		monster, err := g.source.Monster(slug)
		if err != nil {
			return fmt.Errorf("exploration : %w", err)
		}
		if i == 0 || Challenge(monster.ChallengeRating) > Challenge(foe.ChallengeRating) {
			foe = monster
		}
	}
	if _, err := MonsterAttack(foe); err != nil {
		return err
	}
	g.State.Encounter = &Encounter{Monster: foe, HP: foe.HitPoints}
	fmt.Printf("Vous rencontrez %s : %d PV, armure %d.\nUtilisez fight ou flee.\n", foe.Name, foe.HitPoints, foe.ArmorClass)
	return nil
}

func (g *Game) Rest() error {
	if g.State.Encounter != nil {
		return fmt.Errorf("impossible de se reposer pendant un combat")
	}
	if g.State.Rested {
		return fmt.Errorf("vous avez déjà utilisé votre repos pour cette visite")
	}
	g.State.Character.HP = g.State.Character.MaxHP
	g.State.Rested = true
	fmt.Println("Vous retrouvez tous vos points de vie.")
	return nil
}

func (g *Game) CharacterSheet() {
	c := g.State.Character
	fmt.Printf("Niveau : %d\nPV : %d/%d\nXP : %d/%d\nAttaque : +%d\nArmure : %d\nArme : %s (%s + %d)\nOr : %d\nRepos utilisé : %t\n",
		c.Level, c.HP, c.MaxHP, c.XP, c.Level*c.Level*50, c.AttackBonus, c.ArmorClass,
		c.Weapon.Name, c.Weapon.Dice, c.Weapon.Bonus, g.State.Gold, g.State.Rested)
}
