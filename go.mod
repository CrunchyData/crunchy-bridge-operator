module github.com/CrunchyData/crunchy-bridge-operator

go 1.16

require (
	github.com/RHEcosystemAppEng/dbaas-operator v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1 // indirect
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210709001253-0e1f9d693477
	sigs.k8s.io/controller-runtime v0.8.3

)

replace github.com/RHEcosystemAppEng/dbaas-operator v1.0.0 => github.com/RHEcosystemAppEng/dbaas-operator v1.0.1-0.20210727140413-3c5d0fcb7d65
