package game

import (
	"encoding/json"
	"fmt"
	"os"
)

func (g *Game) Save(path string) error {
	data, err := json.MarshalIndent(g.State, "", "  ")
	if err != nil {
		return fmt.Errorf("encodage de la sauvegarde : %w", err)
	}
	if err := os.WriteFile(path+".tmp", data, 0600); err != nil {
		return fmt.Errorf("écriture de la sauvegarde : %w", err)
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		return fmt.Errorf("remplacement de la sauvegarde : %w", err)
	}
	return nil
}

func (g *Game) Load(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("aucune sauvegarde disponible : %w", err)
	}
	if err != nil {
		return fmt.Errorf("lecture de la sauvegarde : %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("décodage de la sauvegarde : %w", err)
	}
	if err := g.validate(state); err != nil {
		return fmt.Errorf("sauvegarde invalide : %w", err)
	}
	g.State = state
	return nil
}

func (g *Game) validate(s State) error {
	if _, ok := g.World.Find(s.Location); !ok {
		return fmt.Errorf("région inconnue : %s", s.Location)
	}
	c := s.Character
	if c.Level < 1 || c.Level > 100 || c.XP < 0 || c.XP > 1000000 {
		return fmt.Errorf("niveau ou expérience invalide")
	}
	if c.MaxHP < 1 || c.MaxHP > 10000 || c.HP < 1 || c.HP > c.MaxHP {
		return fmt.Errorf("points de vie invalides")
	}
	if c.AttackBonus < 0 || c.AttackBonus > 1000 || c.ArmorClass < 1 || c.ArmorClass > 1000 {
		return fmt.Errorf("statistiques du personnage invalides")
	}
	if c.Weapon.Bonus < 0 || c.Weapon.Bonus > 1000 || s.Gold < 0 {
		return fmt.Errorf("équipement ou or invalide")
	}
	if _, _, err := DiceParts(c.Weapon.Dice); err != nil {
		return fmt.Errorf("arme invalide : %w", err)
	}
	if s.Slain == nil {
		return fmt.Errorf("bestiaire des victimes absent")
	}
	for slug, count := range s.Slain {
		if slug == "" || count < 0 {
			return fmt.Errorf("bestiaire des victimes invalide")
		}
	}
	if s.Encounter != nil {
		e := s.Encounter
		if e.Monster.Slug == "" || e.Monster.Name == "" || e.Monster.ArmorClass < 1 {
			return fmt.Errorf("rencontre invalide")
		}
		if e.Monster.HitPoints < 1 || e.HP < 1 || e.HP > e.Monster.HitPoints {
			return fmt.Errorf("points de vie du monstre invalides")
		}
		if _, err := MonsterAttack(e.Monster); err != nil {
			return err
		}
	}
	return nil
}
