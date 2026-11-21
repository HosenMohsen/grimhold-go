package open5e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMonsterDecodingAndCache(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/v1/monsters/goblin/" {
			t.Errorf("chemin inattendu : %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprint(w, `{"slug":"goblin","name":"Goblin","hit_points":7,"armor_class":15,"challenge_rating":"1/4","actions":[{"name":"Scimitar","attack_bonus":4,"damage_dice":"1d6","damage_bonus":2}]}`); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := NewClient(server.URL+"/v1/monsters/", time.Minute)
	defer client.Close()
	for i := 0; i < 2; i++ {
		monster, err := client.Monster("goblin")
		if err != nil {
			t.Fatal(err)
		}
		if monster.HitPoints != 7 || monster.ArmorClass != 15 || monster.ChallengeRating != "1/4" || len(monster.Actions) != 1 || monster.Actions[0].DamageBonus != 2 {
			t.Fatalf("statistiques incorrectes : %+v", monster)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("requêtes : %d, attendu : 1", requests.Load())
	}
	client.mu.Lock()
	entry := client.cache[client.baseURL+"goblin/"]
	entry.Expires = time.Now().Add(-time.Minute)
	client.cache[client.baseURL+"goblin/"] = entry
	client.mu.Unlock()
	if _, err := client.Monster("goblin"); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatal("une réponse expirée a été réutilisée")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := client.Monster("goblin"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestCachePurge(t *testing.T) {
	now := time.Now()
	client := &Client{cache: map[string]cacheEntry{
		"old":   {Expires: now.Add(-time.Second)},
		"limit": {Expires: now},
		"fresh": {Expires: now.Add(time.Minute)},
	}}
	client.purge(now)
	if len(client.cache) != 1 {
		t.Fatalf("entrées restantes : %d, attendu : 1", len(client.cache))
	}
	if _, found := client.cache["fresh"]; !found {
		t.Fatal("une entrée valide a été supprimée")
	}
}

func TestPaginationAndHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/monsters/" {
			http.NotFound(w, r)
			return
		}
		body := `{"next":"?page=2","previous":null,"results":[{"slug":"goblin","name":"Goblin"}]}`
		if r.URL.Query().Get("page") == "2" {
			body = `{"next":null,"previous":"?page=1","results":[{"slug":"skeleton","name":"Skeleton"}]}`
		}
		if _, err := fmt.Fprint(w, body); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := NewClient(server.URL+"/v1/monsters/", time.Minute)
	defer client.Close()
	first, err := client.List("")
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.List(first.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Results) != 1 || second.Results[0].Slug != "skeleton" || second.Previous == "" || second.Next != "" {
		t.Fatalf("pagination incorrecte : %+v", second)
	}
	if _, err := client.Monster("missing"); err == nil {
		t.Fatal("une erreur HTTP doit être signalée")
	}
}
