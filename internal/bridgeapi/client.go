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

type ClientOption func(*Client)

type Client struct {
	apiTarget  *url.URL
	authTarget *url.URL
	log        logr.Logger
	client     *http.Client
	session    *loginManager
}

func NewClient(apiURL *url.URL, cp CredentialProvider, opts ...ClientOption) (*Client, error) {
	if apiURL == nil {
		return nil, errors.New("cannot create client to nil URL target")
	}

	// Defaults unless overridden by options
	c := &Client{
		apiTarget:  apiURL,
		authTarget: apiURL,
		log:        logr.Discard(),
		client:     &http.Client{},
	}

	for _, opt := range opts {
		opt(c)
	}

	if sess, err := sessionCache.GetSession(c.authTarget, cp, c.log); err != nil {
		return nil, err
	} else {
		c.session = sess
	}

	fmt.Printf("Session state: %#v\n", c.session)
	return c, nil
}

// SetAuthURL allows setting a different authentication provider URL if
// different from the API URL, defaults to the API URL provided in NewClient
func SetAuthURL(authURL *url.URL) ClientOption {
	return func(c *Client) {
		c.authTarget = authURL
	}
}

func SetLogger(logger logr.Logger) ClientOption {
	return func(c *Client) {
		c.log = logger
	}
}

// SetHTTPClient allows the use of a custom-configured HTTP client for API
// requests, Client defaults to a default http.Client{} otherwise
func SetHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.client = hc
	}
}

func (c *Client) precheck() error {
	// Attempt to refresh login state if inactive (and not bad creds)
	if c.session != nil {
		c.session.Ping()
	} else {
		return errors.New("nil session - WTF?!")
	}

	return c.GetLoginState().toError()
}

func (c *Client) GetLoginState() LoginState {
	return c.session.State()
}

// helper to set up auth with current bearer token
func (c *Client) setBearer(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.session.token())
}

func (c *Client) CreateCluster(cr CreateRequest) error {
	if err := c.precheck(); err != nil {
		return err
	}
	// TODO: Identify personal team id if not provided in request

	reqPayload := new(bytes.Buffer)
	json.NewEncoder(reqPayload).Encode(cr)
	req, err := http.NewRequest(http.MethodPost, c.apiTarget.String()+routeClusters, reqPayload)
	if err != nil {
		c.log.Error(err, "during create cluster request")
		return err
	}
	c.setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error(err, "during create cluster")
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
		c.log.Info("unrecognized return status from create call", "statusCode", resp.StatusCode)
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

	req, err := http.NewRequest(http.MethodGet, c.apiTarget.String()+routeClusters, nil)
	if err != nil {
		c.log.Error(err, "during list personal clusters request prep")
		return ClusterList{}, err
	}
	c.setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error(err, "during personal cluster list request")
		return ClusterList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Info("unexpected status code from API (cluster list)", "statusCode", resp.StatusCode)
		return ClusterList{}, errors.New("unexpected response status from API")
	}

	var myList ClusterList
	err = json.NewDecoder(resp.Body).Decode(&myList)
	if err != nil {
		c.log.Error(err, "error unmarshaling response body for cluster list")
		return ClusterList{}, err
	}
	return myList, nil
}

func (c *Client) ListTeamClusters(teamID string) (ClusterList, error) {
	if err := c.precheck(); err != nil {
		return ClusterList{}, err
	}

	reqURL := fmt.Sprintf("%s%s?team_id=%s", c.apiTarget, routeClusters, teamID)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		c.log.Error(err, "during list team clusters request prep")
		return ClusterList{}, err
	}
	c.setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error(err, "during team cluster list request")
		return ClusterList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Info("unexpected status code from API (team cluster list)", "statusCode", resp.StatusCode)
		return ClusterList{}, errors.New("unexpected response status from API")
	}

	var teamList ClusterList
	err = json.NewDecoder(resp.Body).Decode(&teamList)
	if err != nil {
		c.log.Error(err, "error unmarshaling response body for cluster list")
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

	req, err := http.NewRequest(http.MethodGet, c.apiTarget.String()+routeTeams, nil)
	if err != nil {
		c.log.Error(err, "during list teams prep")
		return ClusterList{}, err
	}
	c.setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error(err, "during list teams")
		return ClusterList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Info("unexpected status code from API (team list)", "statusCode", resp.StatusCode)
		return ClusterList{}, errors.New("unexpected response status from API")
	}

	var teamList struct {
		Teams []struct {
			ID string `json:"id"`
		}
	}
	err = json.NewDecoder(resp.Body).Decode(&teamList)
	if err != nil && teamList.Teams != nil {
		c.log.Error(err, "error unmarshaling response body for team list")
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

	route := fmt.Sprintf(c.apiTarget.String()+routeDefaultRole, id)

	req, err := http.NewRequest(http.MethodGet, route, nil)
	if err != nil {
		c.log.Error(err, "during cluster role request prep")
		return ConnectionRole{}, err
	}
	c.setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error(err, "during cluster role request")
		return ConnectionRole{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Info("unexpected status code from API(cluster role)", "statusCode", resp.StatusCode)
		return ConnectionRole{}, errors.New("unexpected response status from API")
	}

	var role ConnectionRole
	err = json.NewDecoder(resp.Body).Decode(&role)
	if err != nil {
		c.log.Error(err, "error unmarshaling response body (cluster role)")
		return ConnectionRole{}, err
	}

	return role, nil
}

func (c *Client) ClusterDetail(id string) (ClusterDetail, error) {
	if err := c.precheck(); err != nil {
		return ClusterDetail{}, err
	}

	route := fmt.Sprintf("%s%s/%s", c.apiTarget, routeClusters, id)

	req, err := http.NewRequest(http.MethodGet, route, nil)
	if err != nil {
		c.log.Error(err, "during cluster detail request")
		return ClusterDetail{}, err
	}
	c.setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error(err, "during cluster detail request prep")
		return ClusterDetail{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Info("unexpected status code from API(cluster detail)", "statusCode", resp.StatusCode)
		return ClusterDetail{}, errors.New("unexpected response status from API")
	}

	var detail struct {
		Cluster ClusterDetail
	}
	err = json.NewDecoder(resp.Body).Decode(&detail)
	if err != nil {
		c.log.Error(err, "error unmarshaling response body (cluster detail)")
		return ClusterDetail{}, err
	}
	return detail.Cluster, nil
}

func (c *Client) DeleteCluster(id string) error {
	if err := c.precheck(); err != nil {
		return err
	}

	route := fmt.Sprintf("%s%s/%s", c.apiTarget, routeClusters, id)

	req, err := http.NewRequest(http.MethodDelete, route, nil)
	if err != nil {
		c.log.Error(err, "during cluster delete request")
		return err
	}
	c.setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error(err, "during cluster delete request prep")
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.log.Info("unexpected status code from API(cluster delete)", "statusCode", resp.StatusCode)
		return errors.New("unexpected response status from API")
	}

	return nil
}

// DefaultTeamID returns the team id for creation requests
//   currently retrieves personal team id, future change for configured default
func (c *Client) DefaultTeamID() (string, error) {
	if err := c.precheck(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, c.apiTarget.String()+routeTeams, nil)
	if err != nil {
		c.log.Error(err, "during list teams prep")
		return "", err
	}
	c.setBearer(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error(err, "during list teams")
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Info("unexpected status code from API (team list)", "statusCode", resp.StatusCode)
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
		c.log.Error(err, "error unmarshaling response body for team list")
		return "", err
	}

	for _, team := range teamList.Teams {
		if team.IsPersonal {
			return team.ID, nil
		}
	}
	return "", errors.New("unable to identify personal team")
}
