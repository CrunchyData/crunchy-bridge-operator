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

	"github.com/go-logr/logr"
	"github.com/jpillora/backoff"
)

const (
	// refreshBuffer represents the time to attempt to refresh the login
	// in seconds prior to expiration time
	refreshBuffer = 15
)

// TODO: move login manager from package global to client internal
var primaryLogin *loginManager

type loginManager struct {
	// Not protected by mutex, only set at init
	authTarget  *url.URL
	loginSource CredentialProvider
	label       string

	// Protected via mutex
	sync.RWMutex
	log           logr.Logger
	activeToken   string
	activeTokenID string
	refreshTimer  *time.Timer
	expireTimer   *time.Timer
	retryDelay    backoff.Backoff
	curState      LoginState
	lastRefresh   time.Time
	lastUsage     time.Time
}

func newLoginManager(
	target *url.URL,
	cp CredentialProvider,
	logger logr.Logger,
	lbl string) *loginManager {
	lm := &loginManager{
		label:       lbl,
		loginSource: cp,
		authTarget:  target,
		log:         logr.Discard(),
		lastRefresh: time.Now(),
		lastUsage:   time.Now().Add(time.Millisecond), // Initial condition lastUsage > lastRefresh
		retryDelay: backoff.Backoff{
			Min:    500 * time.Millisecond,
			Max:    3600 * time.Second,
			Factor: 2,
			Jitter: true,
		},
	}
	lm.login()
	return lm
}

func (lm *loginManager) SetLogger(logger logr.Logger) {
	lm.Lock()
	defer lm.Unlock()

	lm.log = logger
}

// Ping attempts to refresh login state when in a temporary non-active state
func (lm *loginManager) Ping() {
	lm.RLock()
	state := lm.curState
	lm.RUnlock()

	if state == LoginFailed || state == LoginInactive || state == LoginUnstarted {
		lm.login()
	}
}

func (lm *loginManager) refreshLogin() {
	refresh := false
	lm.Lock()
	refresh = lm.lastUsage.After(lm.lastRefresh)
	lm.Unlock()
	if refresh {
		lm.login()
	}
}

func (lm *loginManager) login() {
	creds, err := lm.loginSource.ProvideCredential()
	if err != nil {
		lm.log.Error(err, "error retrieving credentials")
		lm.setNextLogin(lm.retryDelay.Duration(), lm.login)
		return
	}
	if creds.Zero() {
		// Fast fail login process for unset credentials, may be expected
		// depending on "eventual consistency" usage
		lm.log.Info("provided credentials currently blank")
		lm.setNextLogin(lm.retryDelay.Duration(), lm.login)
		return
	}

	req, err := http.NewRequest(http.MethodPost, lm.authTarget.String()+"/access-tokens", nil)
	if err != nil {
		lm.log.Error(err, "error creating token login request")
		lm.setNextLogin(lm.retryDelay.Duration(), lm.login)
		lm.failLoginTemp()
		return
	}
	req.SetBasicAuth(creds.Key, creds.Secret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		lm.log.Error(err, "error creating http client")
		lm.setNextLogin(lm.retryDelay.Duration(), lm.login)
		lm.failLoginTemp()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		lm.log.Error(fmt.Errorf("API returned status %d for login [%s]", resp.StatusCode, creds.Key), "login failure")
		lm.Lock()
		lm.curState = LoginInvalidCreds
		lm.Unlock()
		lm.setNextLogin(lm.retryDelay.Duration(), lm.login)
		return
	} else if resp.StatusCode != http.StatusOK {
		lm.log.Error(
			fmt.Errorf("API returned unexpected response %d for login [%s]", resp.StatusCode, creds.Key),
			"unexpected login response")
		lm.setNextLogin(lm.retryDelay.Duration(), lm.login)
		lm.failLoginTemp()
		return
	}

	var tr tokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tr)
	if err != nil {
		lm.log.Error(err, "error unmarshaling token response body")
		lm.setNextLogin(lm.retryDelay.Duration(), lm.login)
		lm.failLoginTemp()
		return
	}

	lm.Lock()
	lm.activeToken = tr.Token
	lm.activeTokenID = tr.TokenID
	lm.curState = LoginActive
	lm.retryDelay.Reset()
	lm.Unlock()

	lm.setExpiration(tr.ExpiresIn)
	lm.setNextLogin(time.Duration(tr.ExpiresIn-refreshBuffer)*time.Second, lm.refreshLogin)
}

func (lm *loginManager) failLoginTemp() {
	lm.Lock()
	defer lm.Unlock()

	if lm.curState == LoginUnstarted {
		lm.curState = LoginFailed
	}
}

func (lm *loginManager) setNextLogin(delay time.Duration, loginFunc func()) {
	lm.Lock()
	defer lm.Unlock()
	// If refresh timer exists, clean it up before creating new
	if lm.refreshTimer != nil {
		lm.refreshTimer.Stop()
	}
	lm.refreshTimer = time.AfterFunc(delay, loginFunc)
}

func (lm *loginManager) expireLogin() {
	lm.Lock()
	defer lm.Unlock()
	lm.activeToken = ""
	if lm.curState == LoginActive {
		lm.curState = LoginInactive
	}
}

func (lm *loginManager) token() string {
	lm.Lock()
	defer lm.Unlock()
	lm.lastUsage = time.Now()

	return lm.activeToken
}

func (lm *loginManager) setExpiration(sec int64) {
	lm.Lock()
	defer lm.Unlock()

	// If expire timer exists, clean it up before creating new
	if lm.expireTimer != nil {
		lm.expireTimer.Stop()
	}
	lm.expireTimer = time.AfterFunc(time.Second*time.Duration(sec), lm.expireLogin)
}

func (lm *loginManager) State() LoginState {
	lm.RLock()
	defer lm.RUnlock()
	return lm.curState
}
