module github.com/CrunchyData/crunchy-bridge-operator

go 1.16

require (
	github.com/RHEcosystemAppEng/dbaas-operator v0.1.0
	github.com/go-logr/logr v0.4.0
	github.com/google/uuid v1.1.2
	github.com/jpillora/backoff v1.0.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/utils v0.0.0-20210709001253-0e1f9d693477
	sigs.k8s.io/controller-runtime v0.9.0

)

replace github.com/RHEcosystemAppEng/dbaas-operator v0.1.0 => github.com/redhatHameed/dbaas-operator v0.0.0-20211222192717-1e4975ae6897
