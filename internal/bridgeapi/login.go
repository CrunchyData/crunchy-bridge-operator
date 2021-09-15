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
	sync.RWMutex
	activeToken   string
	activeTokenID string
	apiTarget     *url.URL
	curState      LoginState
	refreshTimer  *time.Timer
	expireTimer   *time.Timer
	loginSource   CredentialProvider
	retryDelay    backoff.Backoff
}

func newLoginManager(cp CredentialProvider, target *url.URL) *loginManager {
	lm := &loginManager{
		loginSource: cp,
		apiTarget:   target,
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

func (lm *loginManager) login() {
	creds, err := lm.loginSource.ProvideCredential()
	if err != nil {
		pkgLog.Error(err, "error retrieving credentials")
		lm.setNextLogin(lm.retryDelay.Duration())
		return
	}
	if creds.Zero() {
		// Fast fail login process for unset credentials, may be expected
		// depending on "eventual consistency" usage
		pkgLog.Info("provided credentials currently blank")
		lm.setNextLogin(lm.retryDelay.Duration())
		return
	}

	req, err := http.NewRequest(http.MethodPost, lm.apiTarget.String()+"/token", nil)
	if err != nil {
		pkgLog.Error(err, "error creating token login request")
		lm.setNextLogin(lm.retryDelay.Duration())
		lm.failLoginTemp()
		return
	}
	req.SetBasicAuth(creds.Key, creds.Secret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		pkgLog.Error(err, "error creating http client")
		lm.setNextLogin(lm.retryDelay.Duration())
		lm.failLoginTemp()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		pkgLog.Error(fmt.Errorf("API returned status %d for login [%s]", resp.StatusCode, creds.Key), "login failure")
		lm.Lock()
		lm.curState = LoginInvalidCreds
		lm.Unlock()
		lm.setNextLogin(lm.retryDelay.Duration())
		return
	} else if resp.StatusCode != http.StatusOK {
		pkgLog.Error(
			fmt.Errorf("API returned unexpected response %d for login [%s]", resp.StatusCode, creds.Key),
			"unexpected login response")
		lm.setNextLogin(lm.retryDelay.Duration())
		lm.failLoginTemp()
		return
	}

	var tr tokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tr)
	if err != nil {
		pkgLog.Error(err, "error unmarshaling token response body")
		lm.setNextLogin(lm.retryDelay.Duration())
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
	lm.setNextLogin(time.Duration(tr.ExpiresIn-refreshBuffer) * time.Second)
}

func (lm *loginManager) failLoginTemp() {
	lm.Lock()
	defer lm.Unlock()

	if lm.curState == LoginUnstarted {
		lm.curState = LoginFailed
	}
}

func (lm *loginManager) setNextLogin(delay time.Duration) {
	lm.Lock()
	defer lm.Unlock()
	// If refresh timer exists, clean it up before creating new
	if lm.refreshTimer != nil {
		lm.refreshTimer.Stop()
	}
	lm.refreshTimer = time.AfterFunc(delay, lm.login)
}

func (lm *loginManager) expireLogin() {
	lm.Lock()
	defer lm.Unlock()
	lm.activeToken = ""
	if lm.curState == LoginActive {
		lm.curState = LoginInactive
	}
}

func (lm *loginManager) reset() {
	lm.Lock()
	defer lm.Unlock()

	lm.loginSource = LoginCred{}
	if lm.refreshTimer != nil {
		lm.refreshTimer.Stop()
	}
	if lm.expireTimer != nil {
		lm.expireTimer.Stop()
	}
	lm.curState = LoginUnstarted
	lm.activeToken = ""
	lm.activeTokenID = ""
	lm.retryDelay.Reset()
}

func (lm *loginManager) logout() {
	lm.Lock()
	defer lm.Unlock()

	if lm.refreshTimer != nil {
		lm.refreshTimer.Stop()
	}
	if lm.expireTimer != nil {
		lm.expireTimer.Stop()
	}
	lm.curState = LoginInactive
	lm.activeToken = ""
	lm.activeTokenID = ""
	lm.retryDelay.Reset()
}

func (lm *loginManager) token() string {
	lm.RLock()
	defer lm.RUnlock()

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

// Resets the LoginManager state to nearly new, ready for a new SetLogin call
func UnsetLogin() {
	if primaryLogin != nil {
		primaryLogin.reset()
	}
}

// "Logs out" by forgetting login state and resetting timers to await next
// login call
func Logout() {
	if primaryLogin != nil {
		primaryLogin.logout()
	}
}

type tokenResponse struct {
	Token     string `json:"access_token"`
	ExpiresIn int64  `json:"expires_in"`
	TokenID   string `json:"id"`
}
