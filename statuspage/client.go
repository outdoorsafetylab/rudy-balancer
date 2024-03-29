package statuspage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"service/log"
	"service/model"
)

type Client struct {
	Client *http.Client
	APIKey string
}

type DataPoint struct {
	Time  time.Time
	Value float64
}

type dataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

func (c *Client) request(method, uri string, reqData, resData interface{}) error {
	var data io.Reader = nil
	if reqData != nil {
		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(reqData)
		if err != nil {
			return err
		}
		data = &buf
	}
	url := fmt.Sprintf("https://api.statuspage.io%s", uri)
	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", c.APIKey))
	log.Debugf("Making request: %s %s", method, url)
	res, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		str, err := io.ReadAll(res.Body)
		if err == nil {
			log.Debugf("%s", str)
		}
		return errors.New(res.Status)
	}
	if resData != nil {
		err = json.NewDecoder(res.Body).Decode(resData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ListComponents(pageID string) ([]*Component, error) {
	list := make([]*Component, 0)
	err := c.request("GET", fmt.Sprintf("/v1/pages/%s/components", pageID), nil, &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) CreateComponent(pageID, groupId, name string) (*Component, error) {
	data := &struct {
		GroupID   string `json:"group_id"`
		Name      string `json:"name"`
		StartDate string `json:"start_date"`
	}{
		GroupID:   groupId,
		Name:      name,
		StartDate: time.Now().Add(-time.Hour * 24).Format("2006-01-02"),
	}
	comp := &Component{}
	err := c.request("POST", fmt.Sprintf("/v1/pages/%s/components", pageID), &struct {
		Component interface{} `json:"component"`
	}{
		Component: data,
	}, comp)
	if err != nil {
		return nil, err
	}
	return comp, nil
}

func (c *Client) UpdateComponentStatus(comp *Component, status string) error {
	data := struct {
		Status string `json:"status"`
	}{
		Status: status,
	}
	err := c.request("PATCH", fmt.Sprintf("/v1/pages/%s/components/%s", comp.PageID, comp.ID), &struct {
		Component interface{} `json:"component"`
	}{
		Component: data,
	}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateIncident(pageID, componentID, name string, affectedSources []*model.Source) (*Incident, error) {
	urlAndErrors := make([]string, len(affectedSources))
	for i, s := range affectedSources {
		urlAndErrors[i] = fmt.Sprintf("%s => %s", s.URL, s.Error)
	}
	data := &struct {
		Name         string   `json:"name"`
		Status       string   `json:"status"`
		ComponentIDs []string `json:"component_ids"`
		Body         string   `json:"body"`
	}{
		Name:         name,
		Status:       "investigating",
		ComponentIDs: []string{componentID},
		Body:         fmt.Sprintf("Affected URLs: %s", strings.Join(urlAndErrors, ", ")),
	}
	incident := &Incident{}
	err := c.request("POST", fmt.Sprintf("/v1/pages/%s/incidents", pageID), &struct {
		Incident interface{} `json:"incident"`
	}{
		Incident: data,
	}, incident)
	if err != nil {
		return nil, err
	}
	return incident, nil
}

func (c *Client) ResolveIncidents(pageID, componentID string) error {
	incidents, err := c.listUnresolvedIncidents(pageID)
	if err != nil {
		return err
	}
	for _, i := range incidents {
		if len(i.Components) > 0 && i.Components[0].ID == componentID {
			err = c.updateIncidentStatus(i, "resolved")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) AddDataPoint(pageID, metricID string, data *DataPoint) error {
	return c.request("POST", fmt.Sprintf("/v1/pages/%s/metrics/%s/data", pageID, metricID), &struct {
		Data dataPoint `json:"data"`
	}{
		dataPoint{
			Timestamp: data.Time.Unix(),
			Value:     data.Value,
		},
	}, nil)
}

func (c *Client) listUnresolvedIncidents(pageID string) ([]*Incident, error) {
	list := make([]*Incident, 0)
	err := c.request("GET", fmt.Sprintf("/v1/pages/%s/incidents/unresolved", pageID), nil, &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) updateIncidentStatus(incident *Incident, status string) error {
	data := struct {
		Status string `json:"status"`
	}{
		Status: status,
	}
	err := c.request("PATCH", fmt.Sprintf("/v1/pages/%s/incidents/%s", incident.PageID, incident.ID), &struct {
		Incident interface{} `json:"incident"`
	}{
		Incident: data,
	}, nil)
	if err != nil {
		return err
	}
	return nil
}
