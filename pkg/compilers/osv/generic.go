package osv

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/emc-advanced-dev/pkg/errors"
	unikos "github.com/emc-advanced-dev/unik/pkg/os"
	"github.com/emc-advanced-dev/unik/pkg/types"
	unikutil "github.com/emc-advanced-dev/unik/pkg/util"
)

func compileRawImage(params types.CompileImageParams, useEc2Bootstrap bool) (string, error) {
	localFolder, err := ioutil.TempDir(unikutil.UnikTmpDir(), "")
	if err != nil {
		return "", err
	}
	// TODO add no cleaup support
	defer os.RemoveAll(localFolder)
	logrus.Debugf("extracting uploaded files to " + localFolder)
	if err := unikos.ExtractTar(params.SourceTar, localFolder); err != nil {
		return "", err
	}
	cmd := exec.Command("docker", "run", "--rm", "--privileged",
		"-v", "/dev/:/dev/",
		"-v", localFolder+"/:/project_directory/",
		"projectunik/compilers-osv-java",
	)
	if useEc2Bootstrap {
		cmd.Args = append(cmd.Args, "-ec2", "true")
	}
	logrus.WithFields(logrus.Fields{
		"command": cmd.Args,
	}).Debugf("running compilers-osv-java container")
	unikutil.LogCommand(cmd, true)
	err = cmd.Run()
	if err != nil {
		return "", errors.New("failed running compilers-osv-java on "+localFolder, err)
	}

	resultFile, err := ioutil.TempFile(unikutil.UnikTmpDir(), "osv-vmdk")
	if err != nil {
		return "", errors.New("failed to create tmpfile for result", err)
	}
	defer func() {
		if err != nil {
			os.Remove(resultFile.Name())
		}
	}()

	if err := os.Rename(filepath.Join(localFolder, "boot.qcow2"), resultFile.Name()); err != nil {
		return "", errors.New("failed to rename result file", err)
	}
	return resultFile.Name(), nil
}