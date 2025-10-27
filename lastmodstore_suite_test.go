package lastmodstore_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLastmodstore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lastmodstore Suite")
}
