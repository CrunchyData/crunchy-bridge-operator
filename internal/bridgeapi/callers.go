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
	"io/ioutil"
	"net/http"
	"net/url"
)

// Leave open configuration for eventual-consistency scenario
type Client struct {
	APITarget *url.URL
}

// helper to set up auth with current bearer token
func setBearer(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+primaryLogin.activeToken)
}

func (c *Client) CreateCluster(cr CreateRequest) error {
	if c.APITarget == nil {
		return ErrorAPIUnset
	}
	// TODO: Identify personal team id if not provided in request

	reqPayload := new(bytes.Buffer)
	json.NewEncoder(reqPayload).Encode(cr)
	req, err := http.NewRequest(http.MethodPost, c.APITarget.String()+routeClusters, reqPayload)
	if err != nil {
		pkgLog.Error(err, "during create cluster request")
		return err
	}
	setBearer(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		pkgLog.Error(err, "during create cluster")
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
		pkgLog.Info("unrecognized return status from create call: %d", resp.StatusCode)
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
	if c.APITarget == nil {
		return ClusterDetail{}, ErrorAPIUnset
	}

	// TODO: switch to ListAll when implemented
	clustList, err := c.ListClusters()
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
	if c.APITarget == nil {
		return ClusterList{}, ErrorAPIUnset
	}

	req, err := http.NewRequest(http.MethodGet, c.APITarget.String()+routeClusters, nil)
	if err != nil {
		pkgLog.Error(err, "during list personal clusters request")
		return ClusterList{}, err
	}
	setBearer(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		pkgLog.Error(err, "during personal cluster list request prep")
		return ClusterList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pkgLog.Info("unexpected status code from API: %d", resp.StatusCode)
		return ClusterList{}, errors.New("unexpected response status from API")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		pkgLog.Error(err, "error reading response body")
		return ClusterList{}, err
	}

	var myList ClusterList
	err = json.Unmarshal(body, &myList)
	if err != nil {
		pkgLog.Error(err, "error unmarshaling response body")
		return ClusterList{}, err
	}
	return myList, nil
}

func (c *Client) ListTeamClusters(teamID string) (ClusterList, error) {
	if c.APITarget == nil {
		return ClusterList{}, ErrorAPIUnset
	}

	return ClusterList{}, errors.New("stub, unimplemented")
}

// ListAllClusters returns all clusters visible to the user, including both
// personal clusters and team visibility
func (c *Client) ListAllClusters() (ClusterList, error) {
	if c.APITarget == nil {
		return ClusterList{}, ErrorAPIUnset
	}

	return ClusterList{}, errors.New("stub, unimplemented")
}

// // Unknown at this point whether it will require a separate query or come
// //  across as part of ClusterDetail
// func (c *Client) SuperuserCredentials(id string) (u, p string) {

// }

func (c *Client) ClusterDetail(id string) (ClusterDetail, error) {
	if c.APITarget == nil {
		return ClusterDetail{}, ErrorAPIUnset
	}

	route := fmt.Sprintf("%s%s/%s", c.APITarget, routeClusters, id)

	req, err := http.NewRequest(http.MethodGet, route, nil)
	if err != nil {
		pkgLog.Error(err, "during cluster detail request")
		return ClusterDetail{}, err
	}
	setBearer(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		pkgLog.Error(err, "during cluster detail request prep")
		return ClusterDetail{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pkgLog.Info("unexpected status code from API(cluster detail): %d", resp.StatusCode)
		return ClusterDetail{}, errors.New("unexpected response status from API")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		pkgLog.Error(err, "error reading response body (cluster detail)")
		return ClusterDetail{}, err
	}

	var detail struct {
		Cluster ClusterDetail
	}
	err = json.Unmarshal(body, &detail)
	if err != nil {
		pkgLog.Error(err, "error unmarshaling response body (cluster detail)")
		return ClusterDetail{}, err
	}
	return detail.Cluster, nil
}

func (c *Client) DeleteCluster(id string) error {
	if c.APITarget == nil {
		return ErrorAPIUnset
	}

	route := fmt.Sprintf("%s%s/%s", c.APITarget, routeClusters, id)

	req, err := http.NewRequest(http.MethodDelete, route, nil)
	if err != nil {
		pkgLog.Error(err, "during cluster delete request")
		return err
	}
	setBearer(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		pkgLog.Error(err, "during cluster delete request prep")
		return err
	}

	if resp.StatusCode != http.StatusOK {
		pkgLog.Info("unexpected status code from API(cluster delete): %d", resp.StatusCode)
		return errors.New("unexpected response status from API")
	}

	return nil
}
