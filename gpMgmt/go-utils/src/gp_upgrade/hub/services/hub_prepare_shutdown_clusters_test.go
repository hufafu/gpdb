package services_test

import (
	"errors"
	"io/ioutil"
	"os"

	"gp_upgrade/hub/configutils"
	"gp_upgrade/hub/services"
	pb "gp_upgrade/idl"
	"gp_upgrade/utils"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"google.golang.org/grpc"
)

var _ = Describe("PrepareShutdownClusters", func() {
	var (
		stdout *gbytes.Buffer
	)

	BeforeEach(func() {
		stdout, _, _ = testhelper.SetupTestLogger()
	})

	AfterEach(func() {
		utils.System = utils.InitializeSystemFunctions()
	})

	// ignoring the go routine
	It("returns successfully", func() {
		utils.System.RemoveAll = func(s string) error { return nil }
		utils.System.MkdirAll = func(s string, perm os.FileMode) error { return nil }

		reader := configutils.NewReader()
		dir, err := ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())
		conf := &services.HubConfig{
			StateDir: dir,
		}
		hub := services.NewHub(&mockClusterPair{}, &reader, grpc.DialContext, conf)

		_, err = hub.PrepareShutdownClusters(nil, &pb.PrepareShutdownClustersRequest{})
		Expect(err).To(BeNil())

		Eventually(stdout.Contents()).Should(ContainSubstring("starting PrepareShutdownClusters()"))
	})

	It("fails if the cluster configuration setup can't be loaded", func() {
		utils.System.RemoveAll = func(s string) error { return nil }
		utils.System.MkdirAll = func(s string, perm os.FileMode) error { return nil }

		reader := configutils.NewReader()
		dir, err := ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())
		conf := &services.HubConfig{
			StateDir: dir,
		}
		clusterPair := &mockClusterPair{
			InitErr: errors.New("boom"),
		}
		hub := services.NewHub(clusterPair, &reader, grpc.DialContext, conf)

		_, err = hub.PrepareShutdownClusters(nil, &pb.PrepareShutdownClustersRequest{})
		Expect(err).To(MatchError("boom"))

		Eventually(stdout.Contents()).Should(ContainSubstring("starting PrepareShutdownClusters()"))
	})
})

type mockClusterPair struct {
	InitErr error
}

func (c *mockClusterPair) StopEverything(str string) {}
func (c *mockClusterPair) Init(baseDir, oldPath, newPath string) error {
	return c.InitErr
}
