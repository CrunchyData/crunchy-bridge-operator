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

type LoginCred struct {
	Key    string
	Secret string
}

func (lc LoginCred) ProvideCredential() LoginCred {
	return lc
}

func (lc LoginCred) Zero() bool {
	return (lc.Key == "" && lc.Secret == "")
}

type CredentialProvider interface {
	ProvideCredential() LoginCred
}
