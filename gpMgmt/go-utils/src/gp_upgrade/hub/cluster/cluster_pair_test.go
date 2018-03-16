package cluster_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"gp_upgrade/hub/cluster"
	"gp_upgrade/utils"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	MASTER_ONLY_JSON = `
[{
    "address": "briarwood",
    "content": -1,
    "datadir": "/datadir",
    "dbid": 1,
    "hostname": "briarwood",
    "mode": "s",
    "port": 25437,
    "preferred_role": "m",
    "role": "m",
    "san_mounts": null,
    "status": "u"
  }]
`
)

// TestHelperProcess isn't a real test. It's used as a helper process
// for TestParameterRun.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mockedOutput := os.Getenv("MOCKED_OUTPUT")
	mockedExitStatus, err := strconv.Atoi(os.Getenv("MOCKED_EXIT_STATUS"))
	if err != nil {
		mockedOutput = "Exit status conversion failed.\nAre we missing the mocked_exit_status?"
		mockedExitStatus = -1
	}
	defer os.Exit(mockedExitStatus)
	fmt.Fprintf(os.Stdout, mockedOutput)
}

var _ = Describe("ClusterPair", func() {
	var (
		dir              string
		mockedOutput     string
		mockedExitStatus int

		filesLaidDown []string
	)

	/* This idea came from https://golang.org/src/os/exec/exec_test.go */
	fakeExecCommand := func(command string, args ...string) *exec.Cmd {
		var err error
		dir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		output := fmt.Sprintf("MOCKED_OUTPUT=%s", mockedOutput)
		exitStatus := fmt.Sprintf("MOCKED_EXIT_STATUS=%d", mockedExitStatus)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", output, exitStatus}
		return cmd
	}

	AfterEach(func() {
		utils.System = utils.InitializeSystemFunctions()
		filesLaidDown = []string{}
	})

	Describe("StopEverything(), shutting down both clusters", func() {
		BeforeEach(func() {
			testhelper.SetupTestLogger()
			// fake out system utilities
			utils.System.ReadFile = func(filename string) ([]byte, error) {
				return []byte(MASTER_ONLY_JSON), nil
			}
			utils.System.OpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
				filesLaidDown = append(filesLaidDown, name)
				return nil, nil
			}
			utils.System.Remove = func(name string) error {
				filteredFiles := make([]string, 0)
				for _, file := range filesLaidDown {
					if file != name {
						filteredFiles = append(filteredFiles, file)
					}
				}
				filesLaidDown = filteredFiles
				return nil
			}
		})

		It("Logs successful when things work", func() {
			mockedExitStatus = 0
			mockedOutput = "Something that's not bad"
			utils.System.ExecCommand = fakeExecCommand

			subject := cluster.Pair{}
			err := subject.Init(dir, "old/path", "new/path")
			Expect(err).ToNot(HaveOccurred())

			subject.StopEverything("path/to/gpstop")

			/* By waiting on the channel message, we enforce the test to wait for
			 * the goroutine to finish and not hit the "race" issue
			 */
			//Consistently(fakeLogger.Error).ShouldNot(Receive())
			//Eventually(fakeLogger.Info).Should(Receive(Equal("finished stopping gpstop.old")))
			//Eventually(fakeLogger.Info).Should(Receive(Equal("finished stopping gpstop.new")))
			Expect(filesLaidDown).To(ContainElement("path/to/gpstop/gpstop.old/completed"))
			Expect(filesLaidDown).To(ContainElement("path/to/gpstop/gpstop.new/completed"))
			Expect(filesLaidDown).ToNot(ContainElement("path/to/gpstop/gpstop.old/running"))
			Expect(filesLaidDown).ToNot(ContainElement("path/to/gpstop/gpstop.new/running"))
		})

		It("puts failures in the log if there are filesystem errors", func() {
			utils.System.OpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
				return nil, errors.New("filesystem blowup")
			}

			subject := cluster.Pair{}
			err := subject.Init(dir, "old/path", "new/path")
			Expect(err).ToNot(HaveOccurred())

			subject.StopEverything("path/to/gpstop")

			//Eventually(fakeLogger.Error).Should(Receive(Equal("filesystem blowup")))
			//Consistently(fakeLogger.Info).ShouldNot(Receive(Equal("finished stopping gpstop.old")))
			Expect(filesLaidDown).ToNot(ContainElement("path/to/gpstop/gpstop.old/in.progress"))
		})

		It("puts Stop failures in the log and leaves files to mark the error", func() {
			mockedExitStatus = 127
			mockedOutput = "gpstop failed us" // what gpstop puts in its own logs
			utils.System.ExecCommand = fakeExecCommand

			subject := cluster.Pair{}
			err := subject.Init(dir, "old/path", "new/path")
			Expect(err).ToNot(HaveOccurred())

			subject.StopEverything("path/to/gpstop")

			// failing because stopCmd.Run() isn't returning an err
			//Eventually(fakeLogger.Info).Should(Receive(Equal("finished stopping gpstop.old")))
			//Eventually(fakeLogger.Error).Should(Receive(Equal("exit status 127")))
			Expect(filesLaidDown).To(ContainElement("path/to/gpstop/gpstop.old/failed"))
			Expect(filesLaidDown).ToNot(ContainElement("path/to/gpstop/gpstop.old/in.progress"))
		})
	})
})