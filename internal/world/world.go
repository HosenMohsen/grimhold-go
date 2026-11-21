package world

import (
	"encoding/json"
	"fmt"
	"os"
)

type Region struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Danger      int      `json:"danger"`
	Foes        []string `json:"foes"`
	Exits       []string `json:"exits"`
}

type World struct {
	Start   string   `json:"start"`
	Regions []Region `json:"regions"`
}

func Load(path string) (World, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return World{}, fmt.Errorf("lecture du monde : %w", err)
	}
	var w World
	if err := json.Unmarshal(data, &w); err != nil {
		return World{}, fmt.Errorf("lecture du JSON du monde : %w", err)
	}
	ids := make(map[string]bool)
	for _, r := range w.Regions {
		if r.ID == "" || ids[r.ID] || r.Danger < 1 || r.Danger > 5 || len(r.Foes) == 0 {
			return World{}, fmt.Errorf("région invalide : %s", r.ID)
		}
		ids[r.ID] = true
	}
	if !ids[w.Start] || !ids["crossroads"] {
		return World{}, fmt.Errorf("le départ et crossroads doivent exister")
	}
	for _, r := range w.Regions {
		for _, exit := range r.Exits {
			if !ids[exit] {
				return World{}, fmt.Errorf("sortie inconnue : %s", exit)
			}
		}
	}
	return w, nil
}

func (w World) Find(id string) (Region, bool) {
	for _, r := range w.Regions {
		if r.ID == id {
			return r, true
		}
	}
	return Region{}, false
}
