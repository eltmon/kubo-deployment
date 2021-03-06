package kubo_deployment_tests_test

import (
	"path"

	. "github.com/jhvhs/gob-mock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Deploy KuBOSH", func() {
	validGcpEnvironment := path.Join(testEnvironmentPath, "test_gcp")
	validGcpCredsEnvironment := path.Join(testEnvironmentPath, "test_gcp_with_creds")
	validvSphereEnvironment := path.Join(testEnvironmentPath, "test_vsphere")
	validOpenstackEnvironment := path.Join(testEnvironmentPath, "test_openstack")

	JustBeforeEach(func() {
		ApplyMocks(bash, []Gob{Spy("credhub")})
		bash.Source("", func(string) ([]byte, error) {
			return repoDirectoryFunction, nil
		})
	})

	Context("fails", func() {
		BeforeEach(func() {
			boshCli := SpyAndConditionallyCallThrough("bosh-cli", "[[ \"$1\" =~ ^int ]]")
			ApplyMocks(bash, []Gob{boshCli})
		})

		DescribeTable("when wrong number of arguments is used", func(params []string) {
			script := pathToScript("deploy_bosh")
			bash.Source(script, nil)
			code, err := bash.Run("main", params)
			Expect(err).NotTo(HaveOccurred())
			Expect(code).NotTo(Equal(0))
			Expect(stdout).To(gbytes.Say("Usage: "))
		},
			Entry("has no arguments", []string{}),
			Entry("has one argument", []string{"gcp"}),
			Entry("has three arguments", []string{"gcp", "foo", "bar"}),
		)

		It("is given an invalid environment path", func() {
			code, err := bash.Run(pathToScript("deploy_bosh"), []string{pathFromRoot(""), pathFromRoot("README.md")})
			Expect(err).NotTo(HaveOccurred())
			Expect(code).NotTo(Equal(0))
		})

		It("is given a non-existing file", func() {
			code, err := bash.Run(pathToScript("deploy_bosh"), []string{validGcpEnvironment, pathFromRoot("non-existing.file")})
			Expect(err).NotTo(HaveOccurred())
			Expect(code).NotTo(Equal(0))
		})
	})

	Context("succeeds", func() {
		BeforeEach(func() {
			bash.Source(pathToScript("deploy_bosh"), nil)
			boshMock := MockOrCallThrough("bosh-cli", `echo "bosh-cli $@" >&2`, "[ $1 == 'int' ]")
			ApplyMocks(bash, []Gob{boshMock})
		})

		It("runs with a valid environment and an extra file", func() {
			code, err := bash.Run("main", []string{validGcpEnvironment, pathFromRoot("README.md")})
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(0))
		})

		It("deploys to a GCP environment", func() {
			code, err := bash.Run("main", []string{validGcpEnvironment, pathFromRoot("README.md")})
			Expect(stderr).To(gbytes.Say("/bosh-deployment/gcp/cpi.yml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(0))
		})

		It("deploys to a vSphere environment", func() {
			code, err := bash.Run("main", []string{validvSphereEnvironment, pathFromRoot("README.md")})
			Expect(stderr).To(gbytes.Say("/bosh-deployment/vsphere/cpi.yml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(0))
		})

		It("deploys to an Openstack environment", func() {
			code, err := bash.Run("main", []string{validOpenstackEnvironment, pathFromRoot("README.md")})
			Expect(stderr).To(gbytes.Say("/bosh-deployment/openstack/cpi.yml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(0))
		})

		It("interacts with credhub", func() {
			code, err := bash.Run("main", []string{validGcpEnvironment, pathFromRoot("README.md")})
			Expect(stderr).To(gbytes.Say("credhub api --skip-tls-validation -s .+:8844"))
			Expect(stderr).To(gbytes.Say("credhub login -u credhub-user -p "))
			Expect(stderr).To(gbytes.Say("credhub ca-get -n default"))
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(0))
		})

		It("expands the environment path", func() {
			relativePath := testEnvironmentPath + "/../environments/test_gcp"
			code, err := bash.Run("main", []string{relativePath, pathFromRoot("README.md")})
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(0))
			Expect(stderr).NotTo(gbytes.Say("\\.\\./environments/test_gcp"))
		})
	})

	Context("CA generation", func() {
		BeforeEach(func() {
			bash.Source(pathToScript("deploy_bosh"), nil)
		})

		It("runs with an environment", func() {
			code, err := bash.Run("generate_default_ca", []string{validGcpCredsEnvironment})
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(0))
		})

		It("logins to credhub", func() {
			bash.Run("generate_default_ca", []string{validGcpCredsEnvironment})
			Expect(stderr).To(gbytes.Say("credhub api --skip-tls-validation -s internal.ip:8844"))
			Expect(stderr).To(gbytes.Say("credhub login -u credhub-user -p password"))
		})

		It("gets the default CA", func() {
			bash.Run("generate_default_ca", []string{validGcpCredsEnvironment})
			Expect(stderr).To(gbytes.Say("credhub login"))
			Expect(stderr).To(gbytes.Say("credhub ca-get -n default"))
			Expect(stderr).NotTo(gbytes.Say("credhub ca-generate"))
		})

		It("Generates a CA if is isn't found", func() {
			credhubMock := `if [[ "$1" == "ca-get" ]]; then return 1; fi`
			ApplyMocks(bash, []Gob{Mock("credhub", credhubMock)})
			bash.Run("generate_default_ca", []string{validGcpCredsEnvironment})
			Expect(stderr).To(gbytes.Say("credhub ca-get -n default"))
			Expect(stderr).To(gbytes.Say("credhub ca-generate -n default -c internal.ip"))
		})
	})
})
