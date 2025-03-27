package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	config, err := Load("./testdata/hydrus.yml")
	assert.NoError(t, err)

	assert.NotNil(t, config)

	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, int32(2), config.Agent.ChannelManager.MinConf)
}

func TestLoadEnvVariable(t *testing.T) {
	os.Setenv("HYDRUS_CONFIG", "./testdata/hydrus.yml")

	config, err := Load("")
	assert.NoError(t, err)

	assert.NotNil(t, config)

	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, int32(2), config.Agent.ChannelManager.MinConf)
}

func TestLoadError(t *testing.T) {
	os.Setenv("HYDRUS_CONFIG", "invalid")

	_, err := Load("invalid")
	assert.Error(t, err)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Config)
		fail  bool
	}{
		{
			name:  "Invalid allocation percentage",
			setup: func(c *Config) { c.Agent.AllocationPercent = 200 },
			fail:  true,
		},
		{
			name: "Min channel size exceeds max",
			setup: func(c *Config) {
				c.Agent.MinChannelSize = 200000
				c.Agent.MaxChannelSize = 150000
			},
			fail: true,
		},
		{
			name: "Min channel size too low",
			setup: func(c *Config) {
				c.Agent.MinChannelSize = 10_000
			},
			fail: true,
		},
		{
			name:  "Min channels exceeds max channels",
			setup: func(c *Config) { c.Agent.MinChannels = 2; c.Agent.MaxChannels = 1 },
			fail:  true,
		},
		{
			name:  "Invalid channel manager min confirmations",
			setup: func(c *Config) { c.Agent.ChannelManager.MinConf = 0 },
			fail:  true,
		},
		{
			name:  "Invalid channel manager target confirmations",
			setup: func(c *Config) { c.Agent.TargetConf = 1 },
			fail:  true,
		},
		{
			name:  "Negative heuristic weight",
			setup: func(c *Config) { c.Agent.HeuristicWeights.Open.BaseFee = -1 },
			fail:  true,
		},
		{
			name:  "Heuristic weight over one",
			setup: func(c *Config) { c.Agent.HeuristicWeights.Open.FeeRate = 2 },
			fail:  true,
		},
		{
			name:  "Valid configuration",
			setup: validConfig,
			fail:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			if tt.setup != nil {
				tt.setup(c)
			}

			err := c.Validate()
			if tt.fail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func validConfig(c *Config) {
	c.Agent.AllocationPercent = 50
	c.Agent.MinChannelSize = 100000
	c.Agent.MaxChannelSize = 200000
	c.Agent.MinChannels = 1
	c.Agent.MaxChannels = 5
	c.Agent.ChannelManager.MinConf = 1
	c.Agent.TargetConf = 2
	c.Agent.HeuristicWeights.Open = DefaultOpenWeights
	c.Agent.HeuristicWeights.Close = DefaultCloseWeights
	c.Lightning.RPC.MacaroonPath = "./testdata/invoice.macaroon"
	c.Lightning.RPC.TLSCertPath = "./testdata/tls.cert"
}

func TestSetDefaults(t *testing.T) {
	config := &Config{}
	config.setDefaults()

	assert.Equal(t, uint64(60), config.Agent.AllocationPercent)
	assert.Equal(t, uint64(2), config.Agent.MinChannels)
	assert.Equal(t, uint64(200), config.Agent.MaxChannels)
	assert.Equal(t, int32(6), config.Agent.TargetConf)
	assert.Equal(t, uint64(1_000_000), config.Agent.MinChannelSize)
	assert.Equal(t, uint64(10_000_000), config.Agent.MaxChannelSize)
	assert.Equal(t, int32(2), config.Agent.ChannelManager.MinConf, 2)
	assert.Equal(t, uint64(100), config.Agent.ChannelManager.FeeRatePPM)
	assert.Equal(t, uint64(50), config.Agent.ChannelManager.MaxSatvB)
	assert.Equal(t, DefaultOpenWeights, config.Agent.HeuristicWeights.Open)
	assert.Equal(t, DefaultCloseWeights, config.Agent.HeuristicWeights.Close)
	assert.Equal(t, time.Duration(time.Second*30), config.Lightning.RPC.Timeout)
	assert.Equal(t, "info", config.Logging.Level)
}
