/*
Copyright 2022.

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

package controllers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	permsv1beta1 "github.com/infra-mgmt-io/perms/api/v1beta1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

const operatorImage = "example.com/infra-mgmt-io-perms:0.0.1"
const podAppLabel = "app.kubernetes.io/name=perms-controller-manager"
const operatorNamespace = "perms-system"

var err error
var cmd *exec.Cmd

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = permsv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("building the operator image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", operatorImage))
	_, err = Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the the operator image on the cluster")
	err = loadImageToClusterWithName(operatorImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating operator namespace")
	cmd = exec.Command("kubectl", "create", "ns", operatorNamespace)
	_, err = Run(cmd)
	Expect(err).To(Not(HaveOccurred()))

	By("deploying the controller")
	cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", operatorImage))
	_, err = Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("validating that pod is still status.phase=Running")
	getPodStatus := func() error {
		cmd = exec.Command("kubectl", "get",
			"pods", "-l", podAppLabel,
			"-o", "jsonpath={.items[*].status}", "-n", operatorNamespace,
		)
		status, err := Run(cmd)
		fmt.Println(string(status))
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if !strings.Contains(string(status), "\"phase\":\"Running\"") {
			return fmt.Errorf("perms pod in %s status", status)
		}
		return nil
	}
	EventuallyWithOffset(1, getPodStatus, 15*time.Second, time.Second).Should(Succeed())

}, 60)

var _ = AfterSuite(func() {
	By("removing operator namespace")
	cmd := exec.Command("kubectl", "delete", "ns", operatorNamespace)
	_, _ = Run(cmd)

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir
	fmt.Fprintf(GinkgoWriter, "running dir: %s\n", cmd.Dir)

	// To allow make commands be executed from the project directory which is subdir on SDK repo
	// TODO:(user) You might does not need the following code
	if err := os.Chdir(cmd.Dir); err != nil {
		fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/controllers", "", -1)
	return wd, nil
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func loadImageToClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}
