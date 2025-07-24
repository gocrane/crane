package recommend

import (
	"io/ioutil"
	"sync"
	"testing"
	"time"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
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

func TestGetConfig(t *testing.T) {

	analyticsConfig := map[string]string{"resource.cpu-request-percentile": "0.2"}
	configString := "apiVersion: analysis.crane.io/v1alpha1\nkind: ConfigSet\nconfigs:\n  - targets: []\n    properties:\n      resource.cpu-request-percentile: \"0.98\"\n      replicas.workload-min-replicas: \"3\"\n      replicas.pod-min-ready-seconds: \"30\"\n      replicas.pod-available-ratio: \"0.5\"\n      replicas.default-min-replicas: \"3\"\n      replicas.max-replicas-factor: \"3\"\n      replicas.min-cpu-usage-threshold: \"1\"\n      replicas.fluctuation-threshold: \"1.5\"\n      replicas.min-cpu-target-utilization: \"30\"\n      replicas.max-cpu-target-utilization: \"75\"\n      replicas.cpu-target-utilization: \"50\"\n      replicas.cpu-percentile: \"95\"\n      replicas.reference-hpa: \"true\""
	configSet, _ := loadConfigSetFromBytes([]byte(configString))

	var wg sync.WaitGroup

	for i := 1; i < 20; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			time.Sleep(time.Second)
			target := analysisapi.Target{
				Kind:      "Deployment",
				Namespace: "crane-system",
				Name:      "php-apache",
			}

			GetProperties(configSet, target, analyticsConfig)
		}()
	}

	wg.Wait()
}
