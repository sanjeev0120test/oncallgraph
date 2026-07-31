// Package config loads and represents the optional .opsgraph.yaml file
// (legacy .oncallgraph.yaml from the brief rename period is still accepted).
// Everything has sensible defaults so demo/test work with no config at all.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultDataDir is used when config omits data_dir.
const DefaultDataDir = ".opsgraph/data"

// DefaultSince is used when config omits default_since.
const DefaultSince = time.Hour

// Config is the top-level configuration.
type Config struct {
	Version      int                      `yaml:"version"`
	DataDir      string                   `yaml:"data_dir"`
	DefaultSince string                   `yaml:"default_since"`
	Services     map[string]ServiceConfig `yaml:"services"`
	Owners       map[string]OwnerConfig   `yaml:"owners"`
	Connectors   Connectors               `yaml:"connectors"`
	AI           AIConfig                 `yaml:"ai"`
}

// ServiceConfig describes one service's structure (not incident data).
type ServiceConfig struct {
	Aliases   []string  `yaml:"aliases"`
	Owner     string    `yaml:"owner"`
	GitPaths  []string  `yaml:"git_paths"`
	K8s       K8sConfig `yaml:"k8s"`
	Runbook   string    `yaml:"runbook"`
	DependsOn []string  `yaml:"depends_on"`
}

// K8sConfig lists deployments/namespaces relevant to a service.
type K8sConfig struct {
	Deployments []string `yaml:"deployments"`
	Namespaces  []string `yaml:"namespaces"`
}

// OwnerConfig describes an owning team/person.
type OwnerConfig struct {
	Name  string `yaml:"name"`
	Team  string `yaml:"team"`
	Email string `yaml:"email"`
}

// Connectors toggles and configures data sources.
type Connectors struct {
	Fixtures     ToggledConnector `yaml:"fixtures"`
	Git          GitConnector     `yaml:"git"`
	Kubernetes   K8sConnector     `yaml:"kubernetes"`
	Prometheus   URLConnector     `yaml:"prometheus"`
	Alertmanager URLConnector     `yaml:"alertmanager"`
}

// ToggledConnector is a connector with only an enabled flag.
type ToggledConnector struct {
	Enabled bool `yaml:"enabled"`
}

// GitConnector configures the local git connector.
type GitConnector struct {
	Enabled  bool   `yaml:"enabled"`
	RepoPath string `yaml:"repo_path"`
}

// K8sConnector configures the (snapshot-based) kubernetes connector.
type K8sConnector struct {
	Enabled  bool   `yaml:"enabled"`
	Snapshot string `yaml:"snapshot"`
}

// URLConnector is a connector reachable at a URL.
type URLConnector struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

// AIConfig configures the optional local AI layer.
type AIConfig struct {
	Enabled    bool   `yaml:"enabled"`
	OllamaURL  string `yaml:"ollama_url"`
	Model      string `yaml:"model"`
	EmbedModel string `yaml:"embed_model"`
	Timeout    string `yaml:"timeout"`
}

// DefaultConfigCandidates are tried (in order) when Load("") is called.
var DefaultConfigCandidates = []string{".opsgraph.yaml", ".oncallgraph.yaml"}

// Default returns a Config with built-in defaults (used when no file exists).
func Default() *Config {
	return &Config{
		Version:      1,
		DataDir:      DefaultDataDir,
		DefaultSince: "60m",
		Services:     map[string]ServiceConfig{},
		Owners:       map[string]OwnerConfig{},
		Connectors: Connectors{
			Fixtures:   ToggledConnector{Enabled: true},
			Git:        GitConnector{Enabled: true, RepoPath: "."},
			Kubernetes: K8sConnector{Enabled: false}, // enable when snapshot path is set
		},
		AI: AIConfig{
			OllamaURL:  "http://127.0.0.1:11434",
			Model:      "qwen2.5:7b",
			EmbedModel: "nomic-embed-text",
			Timeout:    "20s",
		},
	}
}

// Load reads config from path. If path is empty it tries
// .opsgraph.yaml then legacy .oncallgraph.yaml; if neither exists it
// returns Default(). A malformed file is an error.
func Load(path string) (*Config, error) {
	explicit := path != ""
	candidates := []string{path}
	if !explicit {
		candidates = DefaultConfigCandidates
	}

	var lastNotExist error
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				lastNotExist = err
				continue
			}
			return nil, fmt.Errorf("read config %q: %w", candidate, err)
		}
		cfg := Default()
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config %q: %w", candidate, err)
		}
		cfg.applyDefaults()
		return cfg, nil
	}

	if !explicit && lastNotExist != nil {
		return Default(), nil
	}
	if explicit {
		return nil, fmt.Errorf("read config %q: %w", path, lastNotExist)
	}
	return Default(), nil
}

func (c *Config) applyDefaults() {
	if c.DataDir == "" {
		c.DataDir = DefaultDataDir
	}
	if c.Services == nil {
		c.Services = map[string]ServiceConfig{}
	}
	if c.Owners == nil {
		c.Owners = map[string]OwnerConfig{}
	}
	if c.AI.OllamaURL == "" {
		c.AI.OllamaURL = "http://127.0.0.1:11434"
	}
	if c.AI.Model == "" {
		c.AI.Model = "qwen2.5:7b"
	}
	if c.AI.EmbedModel == "" {
		c.AI.EmbedModel = "nomic-embed-text"
	}
}

// Since parses default_since, falling back to DefaultSince on empty/invalid.
func (c *Config) Since() time.Duration {
	if c.DefaultSince == "" {
		return DefaultSince
	}
	d, err := time.ParseDuration(c.DefaultSince)
	if err != nil || d <= 0 {
		return DefaultSince
	}
	return d
}

// AITimeout parses the AI timeout, defaulting to 20s.
func (c *Config) AITimeout() time.Duration {
	if c.AI.Timeout == "" {
		return 20 * time.Second
	}
	d, err := time.ParseDuration(c.AI.Timeout)
	if err != nil || d <= 0 {
		return 20 * time.Second
	}
	return d
}
