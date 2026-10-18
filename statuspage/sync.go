package statuspage

import (
	"errors"
	"fmt"

	"service/log"
)

// Observation is what one healthcheck run saw of one component.
type Observation struct {
	// Component is the StatusPage component name; it is created in the group
	// when the page has none by that name.
	Component string
	// Status is operational, partial_outage or major_outage.
	Status string
	// Incident names the incident opened when the component stays down.
	Incident string
	// Bad lists the URLs that failed and why, for the incident body.
	Bad map[string]error
}

// SyncPage brings the page in line with what this run observed. It works from
// the page's current state rather than from the change since the last run, so
// a write that failed is retried by the next run instead of being lost. It
// carries on past a failed write and returns every failure, so the run that
// hit one does not end looking successful.
func (c *Client) SyncPage(pageID, groupID string, observed []Observation) error {
	components, err := c.ListComponents(pageID)
	if err != nil {
		return fmt.Errorf("list components: %w", err)
	}
	incidents, err := c.listUnresolvedIncidents(pageID)
	if err != nil {
		return fmt.Errorf("list unresolved incidents: %w", err)
	}
	byName := make(map[string]*Component)
	for _, comp := range components {
		byName[comp.Name] = comp
	}
	open := make(map[string][]*Incident)
	for _, i := range incidents {
		for _, comp := range i.Components {
			open[comp.ID] = append(open[comp.ID], i)
		}
	}
	var errs []error
	for _, o := range observed {
		comp := byName[o.Component]
		if comp == nil {
			log.Warnf("Creating component: %s", o.Component)
			comp, err = c.CreateComponent(pageID, groupID, o.Component)
			if err != nil {
				errs = append(errs, fmt.Errorf("create component %s: %w", o.Component, err))
				continue
			}
		}
		err = c.sync(comp, o, open[comp.ID])
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", o.Component, err))
		}
	}
	return errors.Join(errs...)
}

// nextStatus is the status to show for a component now showing current when
// this run observed observed. A component that was operational is shown as
// degraded_performance on its first failed run and only goes to an outage, and
// gets an incident, if the next run fails too: one timeout should not notify
// the page's subscribers.
func nextStatus(current, observed string) string {
	if observed != "operational" && current == "operational" {
		return "degraded_performance"
	}
	return observed
}

func (c *Client) sync(comp *Component, o Observation, open []*Incident) error {
	status := nextStatus(comp.Status, o.Status)
	if status != comp.Status {
		log.Debugf("Updating component status: %s => %s", o.Component, status)
		err := c.UpdateComponentStatus(comp, status)
		if err != nil {
			return fmt.Errorf("update status to %s: %w", status, err)
		}
	}
	switch {
	case status == "operational":
		var errs []error
		for _, i := range open {
			log.Warnf("Resolving incident: %s is back", o.Component)
			err := c.updateIncidentStatus(i, "resolved")
			if err != nil {
				errs = append(errs, fmt.Errorf("resolve incident %s: %w", i.ID, err))
			}
		}
		return errors.Join(errs...)
	case status != "degraded_performance" && len(open) == 0:
		log.Warnf("Creating incident: %s", o.Incident)
		_, err := c.CreateIncident(comp.PageID, comp.ID, o.Incident, o.Bad)
		if err != nil {
			return fmt.Errorf("create incident: %w", err)
		}
	}
	return nil
}
