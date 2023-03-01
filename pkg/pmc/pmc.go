package pmc

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	expect "github.com/google/goexpect"
	"github.com/openshift/linuxptp-daemon/pkg/dataset"
)

var (
	ClockClassChangeRegEx = regexp.MustCompile(`gm.ClockClass[[:space:]]+(\d+)`)
	CmdParentDataSet      = "GET PARENT_DATA_SET"
	cmdTimeout            = 500 * time.Millisecond
)

// RunPMCExp ... go expect to run PMC util cmd
func RunPMCExp(configFileName, cmdStr string, promptRE *regexp.Regexp) (result string, matches []string, err error) {
	e, _, err := expect.Spawn(fmt.Sprintf("pmc -u -b 1 -f /var/run/%s", configFileName), -1)
	if err != nil {
		return "", []string{}, err
	}
	defer e.Close()
	if err = e.Send(cmdStr + "\n"); err == nil {
		result, matches, err = e.Expect(promptRE, cmdTimeout)
		if err != nil {
			glog.Errorf("pmc result match error %s", err)
			return
		}
		err = e.Send("\x03")
	}
	return
}

// WIP
func GetParentDs(configName string) (result dataset.ParentDs, err error) {

	cmdLine := fmt.Sprintf("pmc -u -b 1 -f /var/run/%s", configName)
	args := strings.Split(cmdLine, " ")
	args = append(args, CmdParentDataSet)
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return dataset.ParentDs{}, err
	}
	ods, err := dataset.ParseParentDataSet(string(out))
	if err != nil {
		return dataset.ParentDs{}, err
	}
	return ods, nil
}
