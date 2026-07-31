package ingest

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sanjeev0120test/oncallgraph/internal/model"
	"github.com/sanjeev0120test/oncallgraph/internal/store"
)

// ServicePaths maps a service id to the repo path prefixes that belong to it.
type ServicePaths struct {
	ServiceID string
	Paths     []string
}

// IngestGit scans a local git repository for commits at/after since and records
// a change per (commit, matching service). It is read-only and offline.
func IngestGit(s *store.Store, repoPath string, services []ServicePaths, since, now time.Time) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("open git repo %q: %w", repoPath, err)
	}
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("git head: %w", err)
	}
	sinceUTC := since.UTC()
	iter, err := repo.Log(&git.LogOptions{From: head.Hash(), Since: &sinceUTC})
	if err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	defer iter.Close()

	return iter.ForEach(func(c *object.Commit) error {
		if c.Author.When.UTC().Before(sinceUTC) {
			return nil
		}
		files, err := changedFiles(c)
		if err != nil {
			return err
		}
		for _, sp := range services {
			if !matchesAny(files, sp.Paths) {
				continue
			}
			if err := recordCommit(s, c, sp.ServiceID); err != nil {
				return err
			}
		}
		return nil
	})
}

func recordCommit(s *store.Store, c *object.Commit, serviceID string) error {
	short := c.Hash.String()[:12]
	evID := "ev-git-" + short + "-" + serviceID
	summary := firstLine(c.Message)
	at := c.Author.When.UTC()
	if err := s.UpsertChange(model.Change{
		ID: "git-" + short + "-" + serviceID, ServiceID: serviceID, At: at, Type: "commit",
		Summary: summary, Author: c.Author.Name, Revision: short, Source: "git", EvidenceID: evID,
	}); err != nil {
		return err
	}
	return s.UpsertEvidence(model.Evidence{
		ID: evID, Source: "git", At: at, Kind: "commit", Summary: summary, RawRef: c.Hash.String(),
	})
}

// changedFiles returns the set of file paths touched by a commit (vs its first
// parent, or the full tree for a root commit).
func changedFiles(c *object.Commit) ([]string, error) {
	stats, err := c.Stats()
	if err != nil {
		return nil, fmt.Errorf("commit stats %s: %w", c.Hash, err)
	}
	out := make([]string, 0, len(stats))
	for _, st := range stats {
		out = append(out, path.Clean(strings.ReplaceAll(st.Name, "\\", "/")))
	}
	sort.Strings(out)
	return out, nil
}

func matchesAny(files, prefixes []string) bool {
	for _, f := range files {
		for _, p := range prefixes {
			p = strings.TrimSuffix(strings.ReplaceAll(p, "\\", "/"), "/")
			if p == "" {
				continue
			}
			if f == p || strings.HasPrefix(f, p+"/") {
				return true
			}
		}
	}
	return false
}

func firstLine(msg string) string {
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		return strings.TrimSpace(msg[:i])
	}
	return strings.TrimSpace(msg)
}
