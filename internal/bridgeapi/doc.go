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

bridgeapi represents a client library to the Crunchy Bridge public API
it structures the API calls in convenient forms and maintains the logged-in
state of the client through the use of internal timers and a provided
source of credential information via the CredentialProvider interface
*/
package bridgeapi
