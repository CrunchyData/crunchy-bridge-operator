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
	"net/url"
	"sync"

	"github.com/go-logr/logr"
)

// Package global cache of loginManager sessions
var sessionCache managerCache

func init() {
	sessionCache = managerCache{
		store: map[string]slot{},
	}
}

type slot struct {
	lm    *loginManager
	count int // Maintain use count to purge unused sessions
}

type managerCache struct {
	sync.RWMutex
	store map[string]slot
}

func (mc *managerCache) GetSession(authURL *url.URL, cp CredentialProvider, logger logr.Logger) (*loginManager, error) {
	label, err := cacheLabel(authURL, cp)
	if err != nil {
		return nil, err
	}
	var lm *loginManager

	mc.Lock()
	defer mc.Unlock()
	if node, ok := mc.store[label]; ok {
		node.count = node.count + 1
		lm = node.lm
		mc.store[label] = node
	} else {
		newNode := slot{
			lm:    newLoginManager(authURL, cp, logger, label),
			count: 1,
		}
		mc.store[label] = newNode
		lm = newNode.lm
	}

	return lm, nil
}

func (mc *managerCache) Release(lm *loginManager) {
	mc.Lock()
	defer mc.Unlock()

	if node, ok := mc.store[lm.label]; ok && node.count > 0 {
		node.count = node.count - 1
		mc.store[lm.label] = node
	}

	// It seems like this could be done on the previous test, but would miss
	// cases where the ref count went to 0 before an unused node reached inactive
	for lbl, node := range mc.store {
		if node.count <= 0 && node.lm.State() == LoginInactive {
			// At this point, the login is both inactive and zero, so the
			// manager would need a whole new login anyway, good time to
			// purge the record and save the resources
			delete(mc.store, lbl)
		}
	}
}

func cacheLabel(authURL *url.URL, cp CredentialProvider) (string, error) {
	cred, err := cp.ProvideCredential()
	if err != nil {
		return "", err
	}
	return authURL.String() + cred.Key, nil
}
