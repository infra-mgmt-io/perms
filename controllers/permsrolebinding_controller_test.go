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

var _ = Describe("permsrolebinding", func() {
	Context("ensure that the operator can handle a CRD in its namespaces", func() {

		It("should successfully run the perms operator", func() {
			projectDir, _ := GetProjectDir()

			By("creating an instance of the PermsRoleBinding CRD in the operator namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsrolebinding_demo2.yaml"), "-n", operatorNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			By("validating that the status of the CR created is updated or not")
			getStatus := func() error {
				cmd = exec.Command("kubectl", "get", "prb",
					"demo2", "-o", "jsonpath={.status.conditions[].status}",
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

		It("it should successfully handle a update for a PermsRoleBinding", func() {
			projectDir, _ := GetProjectDir()

			testNamespace := "testing1"

			By("creating test namespace")
			cmd = exec.Command("kubectl", "create", "ns", testNamespace)
			_, err = Run(cmd)
			Expect(err).To(Not(HaveOccurred()))

			By("creating an instance of the PermsRoleBinding CRD in the test namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsrolebinding_demo2.yaml"), "-n", testNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			time.Sleep(5 * time.Second)

			By("updating and instance of the PermsRoleBinding CRD in the test namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "patch", "prb", "demo2", "--patch-file", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsrolebinding_patch_success.yaml"), "--type", "merge", "-n", testNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			By("validating that the status of the CR created is updated")
			getStatus := func() error {
				cmd = exec.Command("kubectl", "get", "prb",
					"demo2", "-o", "jsonpath={.status.conditions[].status}",
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

		It("it should fail when patching a immutable field of a PermsRoleBinding", func() {
			projectDir, _ := GetProjectDir()

			testNamespace := "testing2"

			By("creating test namespace")
			cmd = exec.Command("kubectl", "create", "ns", testNamespace)
			_, err = Run(cmd)
			Expect(err).To(Not(HaveOccurred()))

			By("creating an instance of the PermsRoleBinding CRD in the testing namespace")
			cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
				"config/samples/perms_v1beta1_permsrolebinding_demo2.yaml"), "-n", testNamespace)
			_, err = Run(cmd)

			time.Sleep(5 * time.Second)

			By("updating the instance of the PermsRoleBinding CRD in the testing namespace")
			cmd = exec.Command("kubectl", "patch", "prb", "demo2", "--patch-file", filepath.Join(projectDir,
				"config/samples/perms_v1beta1_permsrolebinding_patch_error.yaml"), "--type", "merge", "-n", testNamespace)
			_, err = Run(cmd)

			By("validating that the status of the CR created is updated to degraded")
			getStatus := func() error {
				cmd = exec.Command("kubectl", "get", "prb",
					"demo2", "-o", "jsonpath={.status.conditions[2].status}",
					"-n", testNamespace,
				)
				status, err := Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "True") {
					return fmt.Errorf("status condition with type Degraded should be set to true")
				}
				return nil
			}
			Eventually(getStatus, 15*time.Second, time.Second).Should(Succeed())

			By("removing testing namespace")
			cmd = exec.Command("kubectl", "delete", "ns", testNamespace)
			_, _ = Run(cmd)

		})

	})

	Context("ensure that the operator can handle resource in different namespaces", func() {

		It("should successfully handle a PermsRoleBinding in a different namespace", func() {
			projectDir, _ := GetProjectDir()
			testNamespace := "testing3"

			By("creating additional namespace")
			cmd = exec.Command("kubectl", "create", "ns", testNamespace)
			_, err = Run(cmd)
			Expect(err).To(Not(HaveOccurred()))

			By("creating an instance of the PermsRoleBinding CRD in a different namespace")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"config/samples/perms_v1beta1_permsrolebinding_demo2.yaml"), "-n", testNamespace)
				_, err = Run(cmd)
				return err
			}, 15*time.Second, time.Second).Should(Succeed())

			By("validating that the status 'Available' of the CR created is updated or not")
			getStatus := func() error {
				cmd = exec.Command("kubectl", "get", "prb",
					"demo2", "-o", "jsonpath={.status.conditions[0].status}",
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

			By("validating that the status 'Degraded' of the CR created is not updated because there is no error")
			getStatus = func() error {
				cmd = exec.Command("kubectl", "get", "prb",
					"demo2", "-o", "jsonpath={.status.conditions[2].status}",
					"-n", testNamespace,
				)
				status, err := Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "False") {
					return fmt.Errorf("status condition with type Degraded should not be set")
				}
				return nil
			}
			Eventually(getStatus, 15*time.Second, time.Second).Should(Succeed())

			By("removing additional namespace")
			cmd = exec.Command("kubectl", "delete", "ns", testNamespace)
			_, _ = Run(cmd)

		})
	})
})
