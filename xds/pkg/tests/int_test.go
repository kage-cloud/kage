package tests

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type IntTestSuite struct {
	K3sSuite
}

func (i *IntTestSuite) Test() {
	pods, err := i.Kube.CoreV1().Pods("default").List(metav1.ListOptions{})
	if err != nil {
		i.FailNow(err.Error())
	}

	fmt.Println(pods)
}

func TestIntTestSuite(t *testing.T) {
	suite.Run(t, new(IntTestSuite))
}
