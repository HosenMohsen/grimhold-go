package open5e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const BaseURL = "https://api.open5e.com/v1/monsters/"

type Action struct {
	Name        string `json:"name"`
	Description string `json:"desc"`
	AttackBonus int    `json:"attack_bonus"`
	DamageDice  string `json:"damage_dice"`
	DamageBonus int    `json:"damage_bonus"`
}

type Monster struct {
	Slug            string   `json:"slug"`
	Name            string   `json:"name"`
	HitPoints       int      `json:"hit_points"`
	ArmorClass      int      `json:"armor_class"`
	ChallengeRating string   `json:"challenge_rating"`
	Actions         []Action `json:"actions"`
}

type Page struct {
	Next     string    `json:"next"`
	Previous string    `json:"previous"`
	Results  []Monster `json:"results"`
}

type cacheEntry struct {
	Data    []byte
	Expires time.Time
}

type Client struct {
	http    *http.Client
	baseURL string
	ttl     time.Duration
	mu      sync.Mutex
	cache   map[string]cacheEntry
	stop    chan struct{}
	done    chan struct{}
}

func NewClient(baseURL string, ttl time.Duration) *Client {
	c := &Client{
		http:    &http.Client{Timeout: 10 * time.Second},
		baseURL: baseURL,
		ttl:     ttl,
		cache:   make(map[string]cacheEntry),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go c.reap()
	return c
}

func (c *Client) Close() {
	close(c.stop)
	<-c.done
}

func (c *Client) reap() {
	defer close(c.done)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			c.purge(now)
		case <-c.stop:
			return
		}
	}
}

func (c *Client) purge(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.cache {
		if !now.Before(entry.Expires) {
			delete(c.cache, key)
		}
	}
}

func (c *Client) get(address string, result any) error {
	c.mu.Lock()
	entry, found := c.cache[address]
	c.mu.Unlock()
	if found && time.Now().Before(entry.Expires) {
		if err := json.Unmarshal(entry.Data, result); err != nil {
			return fmt.Errorf("décodage du cache : %w", err)
		}
		return nil
	}
	response, err := c.http.Get(address)
	if err != nil {
		return fmt.Errorf("connexion à Open5e : %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, 8*1024*1024))
	closeErr := response.Body.Close()
	if readErr != nil {
		return fmt.Errorf("lecture de la réponse : %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("fermeture de la réponse : %w", closeErr)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Open5e répond avec le statut %d", response.StatusCode)
	}
	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("décodage de la réponse : %w", err)
	}
	c.mu.Lock()
	c.cache[address] = cacheEntry{Data: data, Expires: time.Now().Add(c.ttl)}
	c.mu.Unlock()
	return nil
}

func (c *Client) Monster(slug string) (Monster, error) {
	if slug == "" || strings.ContainsAny(slug, "/?#") {
		return Monster{}, fmt.Errorf("identifiant de monstre invalide")
	}
	var monster Monster
	if err := c.get(c.baseURL+url.PathEscape(slug)+"/", &monster); err != nil {
		return Monster{}, fmt.Errorf("monstre %s : %w", slug, err)
	}
	if monster.Name == "" || monster.HitPoints < 1 || monster.ArmorClass < 1 {
		return Monster{}, fmt.Errorf("statistiques invalides pour %s", slug)
	}
	return monster, nil
}

func (c *Client) List(address string) (Page, error) {
	if address == "" {
		address = c.baseURL + "?limit=10"
	}
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return Page{}, fmt.Errorf("adresse de base : %w", err)
	}
	link, err := url.Parse(address)
	if err != nil {
		return Page{}, fmt.Errorf("adresse de la page : %w", err)
	}
	link = base.ResolveReference(link)
	if link.Scheme != base.Scheme || link.Host != base.Host || link.Path != base.Path {
		return Page{}, fmt.Errorf("adresse de pagination invalide")
	}
	var page Page
	if err := c.get(link.String(), &page); err != nil {
		return Page{}, fmt.Errorf("lecture du bestiaire : %w", err)
	}
	return page, nil
}
