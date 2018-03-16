package integrations_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"gp_upgrade/hub/cluster"
	"gp_upgrade/hub/configutils"
	"gp_upgrade/hub/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"google.golang.org/grpc"
)

var _ = Describe("version command", func() {
	var (
		dir string
		hub *services.HubClient
	)

	BeforeEach(func() {
		var err error
		dir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		conf := &services.HubConfig{
			CliToHubPort:   7527,
			HubToAgentPort: 6416,
			StateDir:       dir,
		}
		reader := configutils.NewReader()
		hub = services.NewHub(&cluster.Pair{}, &reader, grpc.DialContext, conf)

		Expect(checkPortIsAvailable(7527)).To(BeTrue())
		go hub.Start()
	})

	AfterEach(func() {
		hub.Stop()
		Expect(checkPortIsAvailable(7527)).To(BeTrue())
		os.RemoveAll(dir)
	})

	It("reports the version that's injected at build-time", func() {
		fake_version := fmt.Sprintf("v0.0.0-dev.%d", time.Now().Unix())
		commandPathWithVersion, err := Build("gp_upgrade/cli", "-ldflags", "-X gp_upgrade/cli/commanders.GpdbVersion="+fake_version)
		Expect(err).NotTo(HaveOccurred())

		// can't use the runCommand() integration helper function because we calculated a separate path
		cmd := exec.Command(commandPathWithVersion, "version")
		session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(session).Should(Exit(0))
		Consistently(session.Out).ShouldNot(Say("unknown version"))
		Eventually(session.Out).Should(Say("gp_upgrade version")) //scans session.Out buffer beyond the matching tokens
		Eventually(session.Out).Should(Say(fake_version))
	})
})
