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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	// refreshBuffer represents the time to attempt to refresh the login
	// in seconds prior to expiration time
	refreshBuffer = 5
)

// TODO: move login manager from package global to client internal
var primaryLogin *loginManager

type loginManager struct {
	sync.RWMutex
	activeToken  string
	apiTarget    *url.URL
	refreshTimer *time.Timer
	loginSource  CredentialProvider
	Error        error
}

func newLoginManager(cp CredentialProvider, target *url.URL) *loginManager {
	lm := &loginManager{
		loginSource: cp,
		apiTarget:   target,
		Error:       nil,
	}

	if err := lm.login(); err != nil {
		lm.Error = err
	}

	return lm
}

func (lm *loginManager) login() error {
	creds := lm.loginSource.ProvideCredential()
	if creds.Error != nil {
		pkgLog.Error(creds.Error, "error creating token login request")
		return creds.Error
	}
	req, err := http.NewRequest(http.MethodPost, lm.apiTarget.String()+"/token", nil)
	if err != nil {
		pkgLog.Error(err, "error creating token login request")
		return err
	}
	req.SetBasicAuth(creds.Key, creds.Secret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.Status == "401 Unauthorized" || resp.StatusCode != http.StatusOK {
		pkgLog.Error(err, "error creating http client", "resp.Status", resp.Status)
		lm.cleamTimer()
		return fmt.Errorf("error creating http clientt: %w , %s", err, resp.Status)
	}

	defer resp.Body.Close()

	var tr tokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tr)
	if err != nil {
		pkgLog.Error(err, "error unmarshaling token response body")
		return err
	}

	lm.Lock()
	defer lm.Unlock()
	lm.activeToken = tr.Token

	lm.cleamTimer()
	lm.refreshTimer = time.AfterFunc(time.Second*time.Duration(tr.ExpiresIn-refreshBuffer),
		func() { lm.login() })

	return nil
}

func (lm *loginManager) cleamTimer() {
	// If refresh timer exists, clean it up before creating new
	if lm.refreshTimer != nil {
		if !lm.refreshTimer.Stop() {
			// Drain channel before leaving to GC
			<-lm.refreshTimer.C
		}
	}
}

func (lm *loginManager) UpdateLogin(cp CredentialProvider) {
	lm.Lock()
	defer lm.Unlock()

	lm.loginSource = cp
	lm.Error = nil
}

func (lm *loginManager) UpdateAuthURL(target *url.URL) {
	lm.Lock()
	defer lm.Unlock()

	lm.apiTarget = target
	lm.Error = nil
}

func SetLogin(cp CredentialProvider, authBaseURL *url.URL) error {
	if primaryLogin == nil {
		primaryLogin = newLoginManager(cp, authBaseURL)
		if primaryLogin.Error != nil {
			err := primaryLogin.Error
			primaryLogin = nil
			return err
		}
	} else {
		primaryLogin.UpdateLogin(cp)
		primaryLogin.UpdateAuthURL(authBaseURL)
	}

	return nil
}

type tokenResponse struct {
	Token     string `json:"access_token"`
	ExpiresIn int64  `json:"expires_in"`
}
