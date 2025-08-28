package ldap

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

var scheme *runtime.Scheme

func TestLDAP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LDAP Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(openldapv1.AddToScheme(scheme)).To(Succeed())
})
