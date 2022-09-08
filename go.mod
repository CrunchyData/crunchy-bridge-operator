module github.com/CrunchyData/crunchy-bridge-operator

go 1.16

require (
	github.com/RHEcosystemAppEng/dbaas-operator v1.0.1-0.20220829191729-018de64ac56f
	github.com/go-logr/logr v1.2.0
	github.com/jpillora/backoff v1.0.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	go.uber.org/zap v1.19.1
	k8s.io/api v0.23.5
	k8s.io/apiextensions-apiserver v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	sigs.k8s.io/controller-runtime v0.11.2

)

replace github.com/RHEcosystemAppEng/dbaas-operator => github.com/xieshenzh/dbaas-operator v1.0.1-0.20220907182015-12def45fad1d
