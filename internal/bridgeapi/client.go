/*
Copyright 2021 Crunchy Data Solutions, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package bridgeapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"
)

var (
	routeClusters    string = "/clusters"
	routeDefaultRole string = "/clusters/%s/roles/default"
	routeTeams       string = "/teams"
)

// Leave open configuration for eventual-consistency scenario
type Client struct {
	APITarget *url.URL
	Log       logr.Logger
	client    *http.Client
}

// NewClient returns a client initialized with the package logger as the
// default logger and an uninitialized APITarget for late binding
func NewClient() *Client {
	client := &Client{
		Log: pkgLog,
	}
	return client
}

func (c *Client) precheck() error {
	if c.APITarget == nil {
		return ErrorAPIUnset
	}

	// Lazy initialization
	if c.client == nil {
		c.client = &http.Client{}
	}

	// Verify login state
	if ls := c.GetLoginState(); ls == LoginFailed || ls == LoginInactive || ls == LoginUnstarted {
		// Make a login attempt on temp failure states before declaring a failure
		primaryLogin.login()
	}
	return c.GetLoginState().toError()
}

func (c *Client) GetLoginState() LoginState {
	if primaryLogin == nil {
		return LoginUnstarted
	} else {
		return primaryLogin.State()
	}
}

// helper to set up auth with current bearer token
func setBearer(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+primaryLogin.token())
}

func (c *Client) CreateCluster(cr CreateRequest) error {
	if err := c.precheck(); err != nil {
		return err
	}
	// TODO: Identify personal team id if not provided in request

	reqPayload := new(bytes.Buffer)
	json.NewEncoder(reqPayload).Encode(cr)
	req, err := http.NewRequest(http.MethodPost, c.APITarget.String()+routeClusters, reqPayload)
	if err != nil {
		c.Log.Error(err, "during create cluster request")
		return err
	}
	setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Log.Error(err, "during create cluster")
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest:
		return ErrorBadRequest
	case http.StatusConflict:
		return ErrorConflict
	default:
		c.Log.Info("unrecognized return status from create call", "statusCode", resp.StatusCode)
		return nil
	}
}

// ClusterByName returns the cluster detail for the named cluster
// at present, it is syntactic sugar for finding the named cluster in the
// ListAllClusters response and retrieving its detail from the individual
// cluster endpoint. This pivot is required as the cluster list does not
// include the state field
//
// Returns a zero-value ClusterDetail and nil error when not found
func (c *Client) ClusterByName(name string) (ClusterDetail, error) {
	if err := c.precheck(); err != nil {
		return ClusterDetail{}, err
	}

	clustList, err := c.ListAllClusters()
	if err != nil {
		return ClusterDetail{}, err
	}

	for _, cluster := range clustList.Clusters {
		if cluster.Name == name {
			return c.ClusterDetail(cluster.ID)
		}
	}
	return ClusterDetail{}, nil
}

func (c *Client) ListClusters() (ClusterList, error) {
	if err := c.precheck(); err != nil {
		return ClusterList{}, err
	}

	req, err := http.NewRequest(http.MethodGet, c.APITarget.String()+routeClusters, nil)
	if err != nil {
		c.Log.Error(err, "during list personal clusters request prep")
		return ClusterList{}, err
	}
	setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Log.Error(err, "during personal cluster list request")
		return ClusterList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Log.Info("unexpected status code from API (cluster list)", "statusCode", resp.StatusCode)
		return ClusterList{}, errors.New("unexpected response status from API")
	}

	var myList ClusterList
	err = json.NewDecoder(resp.Body).Decode(&myList)
	if err != nil {
		c.Log.Error(err, "error unmarshaling response body for cluster list")
		return ClusterList{}, err
	}
	return myList, nil
}

func (c *Client) ListTeamClusters(teamID string) (ClusterList, error) {
	if err := c.precheck(); err != nil {
		return ClusterList{}, err
	}

	reqURL := fmt.Sprintf("%s%s?team_id=%s", c.APITarget, routeClusters, teamID)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		c.Log.Error(err, "during list team clusters request prep")
		return ClusterList{}, err
	}
	setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Log.Error(err, "during team cluster list request")
		return ClusterList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Log.Info("unexpected status code from API (team cluster list)", "statusCode", resp.StatusCode)
		return ClusterList{}, errors.New("unexpected response status from API")
	}

	var teamList ClusterList
	err = json.NewDecoder(resp.Body).Decode(&teamList)
	if err != nil {
		c.Log.Error(err, "error unmarshaling response body for cluster list")
		return ClusterList{}, err
	}
	return teamList, nil
}

// ListAllClusters returns all clusters visible to the user, including both
// personal clusters and team visibility
func (c *Client) ListAllClusters() (ClusterList, error) {
	if err := c.precheck(); err != nil {
		return ClusterList{}, err
	}

	req, err := http.NewRequest(http.MethodGet, c.APITarget.String()+routeTeams, nil)
	if err != nil {
		c.Log.Error(err, "during list teams prep")
		return ClusterList{}, err
	}
	setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Log.Error(err, "during list teams")
		return ClusterList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Log.Info("unexpected status code from API (team list)", "statusCode", resp.StatusCode)
		return ClusterList{}, errors.New("unexpected response status from API")
	}

	var teamList struct {
		Teams []struct {
			ID string `json:"id"`
		}
	}
	err = json.NewDecoder(resp.Body).Decode(&teamList)
	if err != nil && teamList.Teams != nil {
		c.Log.Error(err, "error unmarshaling response body for team list")
		return ClusterList{}, err
	}

	allClusters := ClusterList{
		Clusters: []ClusterDetail{},
	}
	for _, team := range teamList.Teams {
		toAdd, err := c.ListTeamClusters(team.ID)
		if err != nil {
			return ClusterList{}, err
		}
		allClusters.Clusters = append(allClusters.Clusters, toAdd.Clusters...)
		allClusters.Count += toAdd.Count
	}
	// At the time of this code, the team order is sorted by personal, then
	// other teams and by team name
	// Clusters are sorted by cluster name, so the result of this combination
	// remains stable without needing a local sort on allClusters.Clusters
	//
	// That is, the result items' order will remain stable for comparison
	// purposes unless the underlying DB queries remove the sort ordering

	return allClusters, nil
}

// DefaultConnRole returns the default connection role for the cluster
// identified by id
func (c *Client) DefaultConnRole(id string) (ConnectionRole, error) {
	if err := c.precheck(); err != nil {
		return ConnectionRole{}, err
	}

	route := fmt.Sprintf(c.APITarget.String()+routeDefaultRole, id)

	req, err := http.NewRequest(http.MethodGet, route, nil)
	if err != nil {
		c.Log.Error(err, "during cluster role request prep")
		return ConnectionRole{}, err
	}
	setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Log.Error(err, "during cluster role request")
		return ConnectionRole{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Log.Info("unexpected status code from API(cluster role)", "statusCode", resp.StatusCode)
		return ConnectionRole{}, errors.New("unexpected response status from API")
	}

	var role ConnectionRole
	err = json.NewDecoder(resp.Body).Decode(&role)
	if err != nil {
		c.Log.Error(err, "error unmarshaling response body (cluster role)")
		return ConnectionRole{}, err
	}

	return role, nil
}

func (c *Client) ClusterDetail(id string) (ClusterDetail, error) {
	if err := c.precheck(); err != nil {
		return ClusterDetail{}, err
	}

	route := fmt.Sprintf("%s%s/%s", c.APITarget, routeClusters, id)

	req, err := http.NewRequest(http.MethodGet, route, nil)
	if err != nil {
		c.Log.Error(err, "during cluster detail request")
		return ClusterDetail{}, err
	}
	setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Log.Error(err, "during cluster detail request prep")
		return ClusterDetail{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Log.Info("unexpected status code from API(cluster detail)", "statusCode", resp.StatusCode)
		return ClusterDetail{}, errors.New("unexpected response status from API")
	}

	var detail struct {
		Cluster ClusterDetail
	}
	err = json.NewDecoder(resp.Body).Decode(&detail)
	if err != nil {
		c.Log.Error(err, "error unmarshaling response body (cluster detail)")
		return ClusterDetail{}, err
	}
	return detail.Cluster, nil
}

func (c *Client) DeleteCluster(id string) error {
	if err := c.precheck(); err != nil {
		return err
	}

	route := fmt.Sprintf("%s%s/%s", c.APITarget, routeClusters, id)

	req, err := http.NewRequest(http.MethodDelete, route, nil)
	if err != nil {
		c.Log.Error(err, "during cluster delete request")
		return err
	}
	setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Log.Error(err, "during cluster delete request prep")
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.Log.Info("unexpected status code from API(cluster delete)", "statusCode", resp.StatusCode)
		return errors.New("unexpected response status from API")
	}

	return nil
}

// PersonalTeamID returns the team id for the caller's personal team to use
// as the default in creation requests
func (c *Client) PersonalTeamID() (string, error) {
	if err := c.precheck(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, c.APITarget.String()+routeTeams, nil)
	if err != nil {
		c.Log.Error(err, "during list teams prep")
		return "", err
	}
	setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.Log.Error(err, "during list teams")
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Log.Info("unexpected status code from API (team list)", "statusCode", resp.StatusCode)
		return "", errors.New("unexpected response status from API")
	}

	var teamList struct {
		Teams []struct {
			ID         string `json:"id"`
			IsPersonal bool   `json:"is_personal"`
		}
	}
	err = json.NewDecoder(resp.Body).Decode(&teamList)
	if err != nil && teamList.Teams != nil {
		c.Log.Error(err, "error unmarshaling response body for team list")
		return "", err
	}

	for _, team := range teamList.Teams {
		if team.IsPersonal {
			return team.ID, nil
		}
	}
	return "", errors.New("unable to identify personal team")
}
