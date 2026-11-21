package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rendu-go/grimhold/internal/open5e"
	"github.com/rendu-go/grimhold/internal/world"
)

type fakeSource struct{}

func (fakeSource) Monster(slug string) (open5e.Monster, error) {
	return open5e.Monster{
		Slug: slug, Name: "Rat", HitPoints: 1, ArmorClass: 1, ChallengeRating: "1/2",
		Actions: []open5e.Action{{Name: "Bite", AttackBonus: 100, DamageDice: "1d1"}},
	}, nil
}

func newTestGame() *Game {
	w := world.World{Start: "crossroads", Regions: []world.Region{
		{ID: "crossroads", Danger: 1, Foes: []string{"rat"}, Exits: []string{"marsh"}},
		{ID: "marsh", Danger: 1, Foes: []string{"rat"}, Exits: []string{"crossroads"}},
	}}
	return New(w, fakeSource{})
}

func TestVictoryAndLevelUp(t *testing.T) {
	g := newTestGame()
	if err := g.Explore(); err != nil {
		t.Fatal(err)
	}
	if err := g.Fight(); err != nil {
		t.Fatal(err)
	}
	if g.State.Encounter != nil || g.State.Slain["rat"] != 1 || g.State.Character.XP != 50 || g.State.Character.Level != 2 || g.State.Character.MaxHP != 38 || g.State.Character.AttackBonus != 6 || g.State.Gold < 1 {
		t.Fatalf("victoire incorrecte : %+v", g.State)
	}
}

func TestDefeat(t *testing.T) {
	g := newTestGame()
	if err := g.Travel("marsh"); err != nil {
		t.Fatal(err)
	}
	if err := g.Explore(); err != nil {
		t.Fatal(err)
	}
	g.State.Character.HP = 1
	g.State.Encounter.Monster.ArmorClass = 100
	if err := g.Fight(); err != nil {
		t.Fatal(err)
	}
	if g.State.Location != "crossroads" || g.State.Character.HP != 1 || g.State.Encounter != nil || g.State.Rested || len(g.State.Slain) != 0 {
		t.Fatalf("défaite incorrecte : %+v", g.State)
	}
}

func TestTravelAndRest(t *testing.T) {
	g := newTestGame()
	if err := g.Travel("unknown"); err == nil {
		t.Fatal("déplacement invalide accepté")
	}
	if err := g.Rest(); err != nil {
		t.Fatal(err)
	}
	if err := g.Rest(); err == nil {
		t.Fatal("deux repos pendant la même visite")
	}
	if err := g.Travel("marsh"); err != nil {
		t.Fatal(err)
	}
	if err := g.Rest(); err != nil {
		t.Fatal(err)
	}
	if err := g.Explore(); err != nil {
		t.Fatal(err)
	}
	if err := g.Travel("crossroads"); err == nil {
		t.Fatal("déplacement accepté pendant un combat")
	}
}

func TestSaveLoadAndInvalidSave(t *testing.T) {
	g := newTestGame()
	path := filepath.Join(t.TempDir(), "save.json")
	if err := g.Load(path); err == nil {
		t.Fatal("sauvegarde absente acceptée")
	}
	if err := g.Explore(); err != nil {
		t.Fatal(err)
	}
	g.State.Rested = true
	g.State.Gold = 12
	g.State.Slain["goblin"] = 2
	if err := g.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded := newTestGame()
	if err := loaded.Load(path); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.State, loaded.State) {
		t.Fatal("la sauvegarde ne restitue pas tout l'état")
	}
	if err := g.Save(path); err != nil {
		t.Fatalf("remplacement de la sauvegarde : %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"location":"missing"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loaded.Load(path); err == nil {
		t.Fatal("sauvegarde invalide acceptée")
	}
	if !reflect.DeepEqual(g.State, loaded.State) {
		t.Fatal("un chargement invalide a modifié la partie")
	}
}

func TestDiceAndChallenge(t *testing.T) {
	for _, dice := range []string{"", "1d0", "-1d6", "1000d6", "1d6+2", "abc"} {
		if _, err := Roll(dice); err == nil {
			t.Errorf("dés invalides acceptés : %s", dice)
		}
	}
	if value, err := Roll("2d1"); err != nil || value != 2 {
		t.Fatalf("résultat des dés : %d, erreur : %v", value, err)
	}
	if Challenge("1/4") != 0.25 || Challenge("2") != 2 {
		t.Fatal("indice de danger incorrect")
	}
}
