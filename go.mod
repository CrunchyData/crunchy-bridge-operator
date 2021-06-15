module github.com/CrunchyData/crunchy-bridge-operator

go 1.16

require (
	github.com/RHEcosystemAppEng/dbaas-operator v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/stdr v0.4.0
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3

)

replace github.com/RHEcosystemAppEng/dbaas-operator v1.0.0 => github.com/RHEcosystemAppEng/dbaas-operator v0.0.0-20210603204134-1cfd6ca87ee9
