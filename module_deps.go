package gogh

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/sirkon/errors"
	"github.com/sirkon/jsonexec"
	"github.com/sirkon/message"
)

// GetDependency adds the dependency at the given version to the module
func (m *Module[T]) GetDependency(path, version string) error {
	return m.getDependency(path, version)
}

// GetDependencyLatest adds the latest version of the dependency to the module
func (m *Module[T]) GetDependencyLatest(path string) error {
	return m.getDependency(path, "latest")
}

func (m *Module[T]) getDependency(path, version string) error {
	// if the dependency is already here pass it
	var gomod struct {
		Require []struct {
			Path string
		}
	}
	if err := jsonexec.Run(&gomod, "go", "mod", "edit", "--json"); err != nil {
		return errors.Wrap(err, "get current deps")
	}
	for _, d := range gomod.Require {
		if d.Path == path {
			message.Warningf("dependency %s is already here, passing", path)
			return nil
		}
	}

	cmd := exec.Command("go", "get", path+"@"+version)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if v, ok := m.fixedDeps[path]; ok {
		if vv, err := semver.Parse(strings.TrimPrefix(version, "v")); err != nil {
			message.Errorf("this dependency is fixed at v%s, cannot change to %s", v, version)
			version = "v" + v.String()
		} else if v.NE(vv) {
			message.Errorf("this dependency is fixed at v%s, cannot change to v%s", v, vv)
			version = "v" + v.String()
		}
	}

	if err := cmd.Run(); err != nil {
		data := strings.TrimSpace(stderr.String())
		if data == "" {
			return err
		}

		return errors.Wrapf(errors.Wrap(err, data), "run go get %s@%s", path, version)
	}

	return nil
}
