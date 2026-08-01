package runbook

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

// Verifier evaluates runbook checks against current state in the store, using a
// fixed "now" for deterministic time-based checks.
type Verifier struct {
	store *store.Store
	now   time.Time
}

// NewVerifier builds a Verifier.
func NewVerifier(s *store.Store, now time.Time) *Verifier {
	return &Verifier{store: s, now: now.UTC()}
}

// VerifyService loads the service's runbook from the store and verifies it.
// Returns a result with Status "missing" if there is no runbook.
func (v *Verifier) VerifyService(serviceID string) (model.VerifyResult, error) {
	rb, err := v.store.GetRunbook(serviceID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return model.VerifyResult{ServiceID: serviceID, Status: model.StatusMissing}, nil
		}
		return model.VerifyResult{}, err
	}
	return v.Verify(*rb)
}

// Verify evaluates every step of a parsed runbook.
func (v *Verifier) Verify(rb model.Runbook) (model.VerifyResult, error) {
	res := model.VerifyResult{ServiceID: rb.ServiceID, Path: rb.Path}
	for _, step := range rb.Steps {
		sr, err := v.verifyStep(rb.ServiceID, step)
		if err != nil {
			return model.VerifyResult{}, err
		}
		res.Steps = append(res.Steps, sr)
	}
	res.Status = rollup(res.Steps)
	return res, nil
}

func rollup(steps []model.StepVerifyResult) string {
	if len(steps) == 0 {
		return model.StatusManual
	}
	hasFail, hasStale, hasPass, hasManual := false, false, false, false
	for _, s := range steps {
		switch s.Status {
		case model.StatusFail, model.StatusError:
			hasFail = true
		case model.StatusStale:
			hasStale = true
		case model.StatusPass:
			hasPass = true
		case model.StatusManual:
			hasManual = true
		}
	}
	switch {
	case hasFail:
		return model.StatusFail
	case hasStale:
		return model.StatusStale
	case hasPass:
		return model.StatusPass
	case hasManual:
		return model.StatusManual
	default:
		// Unknown step statuses are authoring bugs, not silent success.
		return model.StatusFail
	}
}

func (v *Verifier) verifyStep(serviceID string, step model.RunbookStep) (model.StepVerifyResult, error) {
	sr := model.StepVerifyResult{Number: step.Number, Text: step.Text, Check: step.Check}

	check := strings.TrimSpace(step.Check)
	if check == "" || check == "manual" {
		sr.Status = model.StatusManual
		sr.Message = "manual step"
		if check == "" {
			sr.Message = "no automated check"
		}
		return sr, nil
	}

	name, arg, _ := strings.Cut(check, ":")
	switch name {
	case "deploy_age_lt", "deploy_age_gt":
		return v.checkDeployAge(serviceID, name, arg, sr)
	case "k8s_deployment_exists":
		return v.checkDeploymentExists(arg, sr)
	case "service_healthy":
		return v.checkServiceHealth(arg, true, sr)
	case "service_unhealthy":
		return v.checkServiceHealth(arg, false, sr)
	case "alert_firing":
		return v.checkAlertFiring(arg, sr)
	default:
		sr.Status = model.StatusError
		sr.Message = fmt.Sprintf("unknown check %q", check)
		return sr, nil
	}
}

func (v *Verifier) checkDeployAge(serviceID, kind, arg string, sr model.StepVerifyResult) (model.StepVerifyResult, error) {
	dur, err := time.ParseDuration(arg)
	if err != nil {
		sr.Status = model.StatusError
		sr.Message = fmt.Sprintf("invalid duration %q", arg)
		return sr, nil
	}
	ch, ok, err := v.store.LatestDeployOrRollout(serviceID)
	if err != nil {
		return sr, err
	}
	if !ok {
		sr.Status = model.StatusStale
		sr.Message = "no deploy/rollout on record"
		return sr, nil
	}
	sr.EvidenceID = ch.EvidenceID
	age := v.now.Sub(ch.At)
	pass := age < dur
	if kind == "deploy_age_gt" {
		pass = age > dur
	}
	if pass {
		sr.Status = model.StatusPass
		sr.Message = fmt.Sprintf("last %s %s ago", ch.Type, roundDur(age))
	} else {
		sr.Status = model.StatusStale
		sr.Message = fmt.Sprintf("last %s %s ago (check expected %s %s)", ch.Type, roundDur(age), cmpWord(kind), arg)
	}
	return sr, nil
}

func (v *Verifier) checkDeploymentExists(name string, sr model.StepVerifyResult) (model.StepVerifyResult, error) {
	// Require explicit rollout evidence for this deployment name. A service with
	// source "kubernetes" is not enough (avoids false pass on unrelated deploy).
	ev, err := v.store.GetEvidence("ev-k8s-rollout-" + name)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return sr, err
	}
	if ev != nil {
		sr.Status = model.StatusPass
		sr.Message = fmt.Sprintf("deployment %q present in snapshot", name)
		sr.EvidenceID = ev.ID
		return sr, nil
	}
	sr.Status = model.StatusStale
	sr.Message = fmt.Sprintf("deployment %q not found in snapshot", name)
	return sr, nil
}

func (v *Verifier) checkServiceHealth(name string, wantHealthy bool, sr model.StepVerifyResult) (model.StepVerifyResult, error) {
	svc, err := v.store.GetServiceByNameOrAlias(name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			sr.Status = model.StatusStale
			sr.Message = fmt.Sprintf("service %q not found", name)
			return sr, nil
		}
		return sr, err
	}
	healthy := svc.Health == model.HealthHealthy
	unhealthy := svc.Health == model.HealthDegraded || svc.Health == model.HealthUnhealthy
	ok := healthy
	if !wantHealthy {
		ok = unhealthy
	}
	if ok {
		sr.Status = model.StatusPass
	} else {
		sr.Status = model.StatusStale
	}
	sr.Message = fmt.Sprintf("%s is %s", name, svc.Health)
	return sr, nil
}

func (v *Verifier) checkAlertFiring(name string, sr model.StepVerifyResult) (model.StepVerifyResult, error) {
	al, ok, err := v.store.FindFiringAlert(name)
	if err != nil {
		return sr, err
	}
	if ok {
		sr.Status = model.StatusPass
		status := al.Status
		if status == "" {
			status = "firing"
		}
		sr.Message = fmt.Sprintf("alert %s is %s", al.Name, status)
		sr.EvidenceID = al.EvidenceID
	} else {
		sr.Status = model.StatusStale
		sr.Message = fmt.Sprintf("alert %q is not active (firing/pending)", name)
	}
	return sr, nil
}

func cmpWord(kind string) string {
	if kind == "deploy_age_gt" {
		return ">"
	}
	return "<"
}

func roundDur(d time.Duration) string {
	if d < time.Minute {
		return strconv.Itoa(int(d.Seconds())) + "s"
	}
	return strconv.Itoa(int(d.Minutes())) + "m"
}
