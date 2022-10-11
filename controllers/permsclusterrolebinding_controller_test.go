package controllers

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("permsclusterrolebinding", func() {
	Context("ensure that the operator can run in its namespaces", func() {

		It("should successfully run the perms operator", func() {
			projectDir, _ := GetProjectDir()

			By("creating an instance of the PermsClusterRoleBinding CRD in the operator namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsclusterrolebinding_demo1.yaml"), "-n", operatorNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			By("validating that the status of the CR created is updated")
			getStatus := func() error {
				cmd = exec.Command("kubectl", "get", "pcrb",
					"demo1", "-o", "jsonpath={.status.conditions[].status}",
					"-n", operatorNamespace,
				)
				status, err := Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "True") {
					return fmt.Errorf("status condition with type Available should be set")
				}
				return nil
			}
			Eventually(getStatus, 15*time.Second, time.Second).Should(Succeed())
		})
	})

	Context("ensure that the operator can perform updates on existing resources", func() {

		It("should successfully run the perms operator", func() {
			projectDir, _ := GetProjectDir()

			testNamespace := "testing4"

			By("creating test namespace")
			cmd = exec.Command("kubectl", "create", "ns", testNamespace)
			_, err = Run(cmd)
			Expect(err).To(Not(HaveOccurred()))

			By("creating an instance of the PermsClusterRoleBinding CRD in a different namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsclusterrolebinding_demo1.yaml"), "-n", testNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			time.Sleep(5 * time.Second)

			By("updating and instance of the PermClustersRoleBinding CRD in the test namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "patch", "pcrb", "demo1", "--patch-file", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsclusterrolebinding_patch_success.yaml"), "--type", "merge", "-n", testNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			By("validating that the status of the CR created is updated")
			getStatus := func() error {
				cmd = exec.Command("kubectl", "get", "pcrb",
					"demo1", "-o", "jsonpath={.status.conditions[].status}",
					"-n", testNamespace,
				)
				status, err := Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "True") {
					return fmt.Errorf("status condition with type Available should be set")
				}
				return nil
			}
			Eventually(getStatus, 15*time.Second, time.Second).Should(Succeed())

			By("removing test namespace")
			cmd = exec.Command("kubectl", "delete", "ns", testNamespace)
			_, _ = Run(cmd)

		})
	})

	Context("ensure that the operator does not change immutable fields", func() {

		It("should successfully run the perms operator", func() {
			projectDir, _ := GetProjectDir()

			testNamespace := "testing5"

			By("creating test namespace")
			cmd = exec.Command("kubectl", "create", "ns", testNamespace)
			_, err = Run(cmd)
			Expect(err).To(Not(HaveOccurred()))

			By("creating an instance of the PermsClusterRoleBinding CRD in a different namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsclusterrolebinding_demo1.yaml"), "-n", testNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			time.Sleep(5 * time.Second)

			By("updating and instance of the PermClustersRoleBinding CRD in the test namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "patch", "pcrb", "demo1", "--patch-file", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsclusterrolebinding_patch_error.yaml"), "--type", "merge", "-n", testNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			By("validating that the status of the CR patched is degraded")
			getStatus := func() error {
				cmd = exec.Command("kubectl", "get", "pcrb",
					"demo1", "-o", "jsonpath={.status.conditions[2].status}",
					"-n", testNamespace,
				)
				status, err := Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "True") {
					return fmt.Errorf("status condition with type Degraded should be set")
				}
				return nil
			}
			Eventually(getStatus, 15*time.Second, time.Second).Should(Succeed())

			By("removing test namespace")
			cmd = exec.Command("kubectl", "delete", "ns", testNamespace)
			_, _ = Run(cmd)

		})
	})

})
