package game

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/rendu-go/grimhold/internal/open5e"
)

func DiceParts(dice string) (int, int, error) {
	parts := strings.Split(dice, "d")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("dés invalides : %s", dice)
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("nombre de dés : %w", err)
	}
	sides, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("faces du dé : %w", err)
	}
	if count < 1 || count > 100 || sides < 1 || sides > 1000 {
		return 0, 0, fmt.Errorf("dés hors limites : %s", dice)
	}
	return count, sides, nil
}

func Roll(dice string) (int, error) {
	count, sides, err := DiceParts(dice)
	if err != nil {
		return 0, err
	}
	total := 0
	for i := 0; i < count; i++ {
		total += rand.IntN(sides) + 1
	}
	return total, nil
}

func Challenge(rating string) float64 {
	parts := strings.Split(rating, "/")
	if len(parts) > 2 {
		return 0
	}
	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	if len(parts) == 2 {
		divisor, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || divisor <= 0 {
			return 0
		}
		value /= divisor
	}
	return value
}

func MonsterAttack(monster open5e.Monster) (open5e.Action, error) {
	for _, action := range monster.Actions {
		if action.DamageDice == "" {
			continue
		}
		if _, _, err := DiceParts(action.DamageDice); err == nil {
			return action, nil
		}
	}
	return open5e.Action{}, fmt.Errorf("%s ne possède aucune attaque avec des dés utilisables", monster.Name)
}

func (g *Game) Fight() error {
	e := g.State.Encounter
	if e == nil {
		return fmt.Errorf("aucun monstre à combattre : utilisez explore")
	}
	c := &g.State.Character
	damage, err := Roll(c.Weapon.Dice)
	if err != nil {
		return fmt.Errorf("attaque du personnage : %w", err)
	}
	attack := rand.IntN(20) + 1 + c.AttackBonus
	if attack >= e.Monster.ArmorClass {
		damage = max(1, damage+c.Weapon.Bonus)
		e.HP = max(0, e.HP-damage)
		fmt.Printf("Vous touchez (%d) : %d dégâts. %s : %d PV.\n", attack, damage, e.Monster.Name, e.HP)
	} else {
		fmt.Printf("Votre attaque échoue (%d).\n", attack)
	}
	if e.HP == 0 {
		g.victory()
		return nil
	}
	return g.enemyTurn()
}

func (g *Game) enemyTurn() error {
	e := g.State.Encounter
	action, err := MonsterAttack(e.Monster)
	if err != nil {
		return err
	}
	damage, err := Roll(action.DamageDice)
	if err != nil {
		return fmt.Errorf("attaque du monstre : %w", err)
	}
	attack := rand.IntN(20) + 1 + action.AttackBonus
	if attack >= g.State.Character.ArmorClass {
		damage = max(1, damage+action.DamageBonus)
		g.State.Character.HP = max(0, g.State.Character.HP-damage)
		fmt.Printf("%s utilise %s (%d) : %d dégâts. Vos PV : %d.\n", e.Monster.Name, action.Name, attack, damage, g.State.Character.HP)
	} else {
		fmt.Printf("%s manque son attaque (%d).\n", e.Monster.Name, attack)
	}
	if g.State.Character.HP == 0 {
		g.State.Location = "crossroads"
		g.State.Character.HP = 1
		g.State.Rested = false
		g.State.Encounter = nil
		fmt.Println("Vous vous réveillez au Carrefour avec 1 PV.")
	}
	return nil
}

func (g *Game) Flee() error {
	if g.State.Encounter == nil {
		return fmt.Errorf("aucun combat à fuir")
	}
	if rand.IntN(20)+1+2 >= 12 {
		g.State.Encounter = nil
		fmt.Println("Vous parvenez à fuir.")
		return nil
	}
	fmt.Println("La fuite échoue. Le monstre vous attaque.")
	return g.enemyTurn()
}

func (g *Game) victory() {
	monster := g.State.Encounter.Monster
	xp := max(10, int(Challenge(monster.ChallengeRating)*100))
	gold := rand.IntN(6) + 1
	g.State.Slain[monster.Slug]++
	g.State.Gold += gold
	g.State.Character.XP += xp
	g.State.Encounter = nil
	fmt.Printf("%s est vaincu. Vous gagnez %d XP et %d pièces d'or.\n", monster.Name, xp, gold)
	c := &g.State.Character
	for c.XP >= c.Level*c.Level*50 {
		c.Level++
		c.MaxHP += 8
		c.HP = c.MaxHP
		c.AttackBonus++
		fmt.Printf("Vous passez au niveau %d.\n", c.Level)
	}
}
