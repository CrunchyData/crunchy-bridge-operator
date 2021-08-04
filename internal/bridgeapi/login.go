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
}

func newLoginManager(cp CredentialProvider, target *url.URL) *loginManager {
	lm := &loginManager{
		loginSource: cp,
		apiTarget:   target,
	}

	lm.login()
	return lm
}

func (lm *loginManager) login() {
	creds := lm.loginSource.ProvideCredential()
	req, err := http.NewRequest(http.MethodPost, lm.apiTarget.String()+"/token", nil)
	if err != nil {
		pkgLog.Error(err, "error creating token login request")
		return
	}
	req.SetBasicAuth(creds.Key, creds.Secret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		pkgLog.Error(err, "error creating http client")
		return
	}
	defer resp.Body.Close()

	var tr tokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tr)
	if err != nil {
		pkgLog.Error(err, "error unmarshaling token response body")
		return
	}

	lm.Lock()
	defer lm.Unlock()
	lm.activeToken = tr.Token

	// If refresh timer exists, clean it up before creating new
	if lm.refreshTimer != nil {
		if !lm.refreshTimer.Stop() {
			// Drain channel before leaving to GC
			<-lm.refreshTimer.C
		}
	}
	lm.refreshTimer = time.AfterFunc(time.Second*time.Duration(tr.ExpiresIn-refreshBuffer), lm.login)
}

func (lm *loginManager) UpdateLogin(cp CredentialProvider) {
	lm.Lock()
	defer lm.Unlock()

	lm.loginSource = cp
}

func (lm *loginManager) UpdateAuthURL(target *url.URL) {
	lm.Lock()
	defer lm.Unlock()

	lm.apiTarget = target
}

func SetLogin(cp CredentialProvider, authBaseURL *url.URL) {
	if primaryLogin == nil {
		primaryLogin = newLoginManager(cp, authBaseURL)
	} else {
		primaryLogin.UpdateLogin(cp)
		primaryLogin.UpdateAuthURL(authBaseURL)
	}
}

type tokenResponse struct {
	Token     string `json:"access_token"`
	ExpiresIn int64  `json:"expires_in"`
}
