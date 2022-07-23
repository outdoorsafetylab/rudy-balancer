package statuspage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type Client struct {
	Client *http.Client
	APIKey string
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
	if res.StatusCode < 300 {
		str, err := ioutil.ReadAll(res.Body)
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
	comp := &Component{}
	data := &struct {
		GroupID   string `json:"group_id"`
		Name      string `json:"name"`
		StartDate string `json:"start_date"`
	}{
		GroupID:   groupId,
		Name:      name,
		StartDate: time.Now().Add(-time.Hour * 24).Format("2006-01-02"),
	}
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
		// StartDate string `json:"start_date"`
	}{
		Status: status,
		// StartDate: "2022-07-22",
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
