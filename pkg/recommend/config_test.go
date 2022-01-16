package recommend

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

const configDef = `
apiVersion: analysis.crane.io/v1alpha1
kind: ConfigSet
configs:
- targets:
  - namespace: default
    kind: Deployment
    name: test
  properties:
      kx: vx
      ky: vy
- targets: []
  properties:
    k1: v1
    k2: v2
`

func TestLoadConfigSetFromFile(t *testing.T) {
	f := writeConfig(t, configDef)

	configSet, err := LoadConfigSetFromFile(f)
	assert.NoError(t, err)
	assert.NotNil(t, configSet)
	assert.Equal(t, 2, len(configSet.Configs))

	c := configSet.Configs[0]
	assert.Equal(t, 1, len(c.Targets))
	ta := c.Targets[0]
	assert.Equal(t, "default", ta.Namespace)
	assert.Equal(t, "Deployment", ta.Kind)
	assert.Equal(t, "test", ta.Name)
	assert.Equal(t, 2, len(c.Properties))
	assert.Equal(t, "vx", c.Properties["kx"])
	assert.Equal(t, "vy", c.Properties["ky"])

	c = configSet.Configs[1]
	assert.Equal(t, 0, len(c.Targets))
	assert.Equal(t, 2, len(c.Properties))
	assert.Equal(t, "v1", c.Properties["k1"])
	assert.Equal(t, "v2", c.Properties["k2"])
}

func writeConfig(t *testing.T, config string) string {
	f, err := ioutil.TempFile("", "config.yaml")
	assert.NoError(t, err)

	_, err = f.WriteString(config)
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	return f.Name()
}
