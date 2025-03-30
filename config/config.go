package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/aftermath2/hydrus/logger"

	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
	"gopkg.in/yaml.v3"
)

var (
	// DefaultOpenWeights contains the default values for the channel opening heuristic weights.
	DefaultOpenWeights = OpenWeights{
		Capacity: 1,
		Features: 1,
		Hybrid:   0.8,
		Centrality: CentralityWeights{
			Degree:      0.4,
			Betweenness: 0.8,
			Eigenvector: 0.5,
			Closeness:   0.8,
		},
		Channels: ChannelsWeights{
			BaseFee:        1,
			FeeRate:        0.7,
			InboundBaseFee: 0.8,
			InboundFeeRate: 0.7,
			MinHTLC:        1,
			MaxHTLC:        0.6,
			BlockHeight:    0.8,
		},
	}

	// DefaultCloseWeights contains the default values for the channel closing heuristic weights.
	DefaultCloseWeights = CloseWeights{
		Capacity:       0.5,
		Active:         1,
		NumForwards:    0.8,
		ForwardsAmount: 1,
		Fees:           1,
		PingTime:       0.4,
		BlockHeight:    0.6,
		FlapCount:      0.2,
	}
)

// Config represents the configuration for the application.
type Config struct {
	Lightning Lightning `yaml:"lightning"`
	Agent     Agent     `yaml:"agent"`
	Logging   Logging   `yaml:"logging"`
}

// Agent configuration.
type Agent struct {
	DryRun            bool              `yaml:"dry_run"`
	AllowForceCloses  bool              `yaml:"allow_force_closes"`
	Blocklist         []string          `yaml:"blocklist"`
	Keeplist          []string          `yaml:"keeplist"`
	ChannelManager    ChannelManager    `yaml:"channel_manager"`
	HeuristicWeights  HeuristicsWeights `yaml:"heuristic_weights"`
	AllocationPercent uint64            `yaml:"allocation_percent"`
	MinBatchSize      uint64            `yaml:"min_batch_size"`
	MinChannels       uint64            `yaml:"min_channels"`
	MaxChannels       uint64            `yaml:"max_channels"`
	MinChannelSize    uint64            `yaml:"min_channel_size"`
	MaxChannelSize    uint64            `yaml:"max_channel_size"`
	TargetConf        int32             `yaml:"target_conf"`
}

// ChannelManager configuration.
type ChannelManager struct {
	MaxSatvB    uint64 `yaml:"max_sat_vb"`
	MinConf     int32  `yaml:"min_conf"`
	BaseFeeMsat uint64 `yaml:"base_fee_msat"`
	FeeRatePPM  uint64 `yaml:"fee_rate_ppm"`
}

// HeuristicsWeights configuration.
type HeuristicsWeights struct {
	Close CloseWeights `yaml:"close"`
	Open  OpenWeights  `yaml:"open"`
}

// CloseWeights configuration.
type CloseWeights struct {
	Capacity       float64 `yaml:"capacity"`
	Active         float64 `yaml:"active"`
	NumForwards    float64 `yaml:"num_forwards"`
	ForwardsAmount float64 `yaml:"forwards_amount"`
	Fees           float64 `yaml:"fees"`
	BlockHeight    float64 `yaml:"block_height"`
	PingTime       float64 `yaml:"ping_time"`
	FlapCount      float64 `yaml:"flap_count"`
}

// OpenWeights configuration.
type OpenWeights struct {
	Capacity   float64           `yaml:"capacity"`
	Features   float64           `yaml:"features"`
	Hybrid     float64           `yaml:"hybrid"`
	Centrality CentralityWeights `yaml:"centrality"`
	Channels   ChannelsWeights   `yaml:"channels"`
}

// CentralityWeights configuration.
type CentralityWeights struct {
	Degree      float64 `yaml:"degree"`
	Betweenness float64 `yaml:"betweenness"`
	Eigenvector float64 `yaml:"eigenvector"`
	Closeness   float64 `yaml:"closeness"`
}

// ChannelsWeights configuration.
type ChannelsWeights struct {
	BaseFee        float64 `yaml:"base_fee"`
	FeeRate        float64 `yaml:"fee_rate"`
	InboundBaseFee float64 `yaml:"inbound_base_fee"`
	InboundFeeRate float64 `yaml:"inbound_fee_rate"`
	MinHTLC        float64 `yaml:"min_htlc"`
	MaxHTLC        float64 `yaml:"max_htlc"`
	BlockHeight    float64 `yaml:"block_height"`
}

// Lightning configuration.
type Lightning struct {
	RPC RPC `yaml:"rpc"`
}

// Logging configuration.
type Logging struct {
	Level string `yaml:"level"`
}

// RPC configuration.
type RPC struct {
	Address      string        `yaml:"address"`
	TLSCertPath  string        `yaml:"tls_cert_path"`
	MacaroonPath string        `yaml:"macaroon_path"`
	Timeout      time.Duration `yaml:"timeout"`
}

// Load returns a configuration object loaded from a file.
func Load(path string) (*Config, error) {
	if path == "" {
		path = os.Getenv("HYDRUS_CONFIG")
		if path == "" {
			dir, err := os.UserHomeDir()
			if err != nil {
				return nil, errors.Wrap(err, "getting home directory")
			}
			path = filepath.Join(dir, "hydrus.yml")
		}
	}

	f, err := os.OpenFile(path, os.O_RDONLY, 0o600)
	if err != nil {
		return nil, errors.Wrap(err, "opening file")
	}
	defer f.Close()

	var config *Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, errors.Wrap(err, "decoding configuration")
	}

	config.setDefaults()

	level, err := logger.LevelFromString(strings.ToLower(config.Logging.Level))
	if err != nil {
		return nil, err
	}

	logger.SetLoggingLevel(level)

	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	return config, nil
}

// Validate returns an error if the configuration is not valid.
func (c *Config) Validate() error {
	if c.Agent.AllocationPercent <= 0 || c.Agent.AllocationPercent > 100 {
		return errors.Errorf("invalid allocation percentage %d, it must be between 0 and 100",
			c.Agent.AllocationPercent,
		)
	}

	if c.Agent.MinChannelSize < 20_000 {
		return errors.New("minimum channel size must be greater than 20,000 satoshis")
	}

	if c.Agent.MinChannelSize > c.Agent.MaxChannelSize {
		return errors.New("minimum channel size is higher than the maximum value")
	}

	if c.Agent.MinChannels > c.Agent.MaxChannels {
		return errors.New("minimum number of channels is higher than the maximum value")
	}

	if c.Agent.ChannelManager.MinConf == 0 {
		return errors.New("invalid channel manager transcations minimum confirmations")
	}

	if c.Agent.TargetConf < 2 {
		return errors.New("target confirmations must be greater than 1")
	}

	err := IterWeights(c.Agent.HeuristicWeights.Open, func(weight float64) error {
		if weight < 0 {
			return errors.New("heuristic weigths must be equal to or higher than zero")
		}
		if weight > 1 {
			return errors.New("heuristic weigths must be equal to or lower than one")
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = IterWeights(c.Agent.HeuristicWeights.Close, func(weight float64) error {
		if weight < 0 {
			return errors.New("heuristic weigths must be equal to or higher than zero")
		}
		if weight > 1 {
			return errors.New("heuristic weigths must be equal to or lower than one")
		}

		return nil
	})
	if err != nil {
		return err
	}

	if _, err := credentials.NewClientTLSFromFile(c.Lightning.RPC.TLSCertPath, ""); err != nil {
		return errors.Wrap(err, "invalid tls certificate path")
	}

	macBytes, err := os.ReadFile(c.Lightning.RPC.MacaroonPath)
	if err != nil {
		return errors.Wrap(err, "macaroon file missing")
	}

	mac := &macaroon.Macaroon{}
	if err := mac.UnmarshalBinary(macBytes); err != nil {
		return errors.Wrap(err, "invalid macaroon encoding")
	}

	return nil
}

func (c *Config) setDefaults() {
	if c.Agent.AllocationPercent == 0 {
		c.Agent.AllocationPercent = 60
	}

	if c.Agent.MinChannels == 0 {
		c.Agent.MinChannels = 2
	}

	if c.Agent.MaxChannels == 0 {
		c.Agent.MaxChannels = 200
	}

	if c.Agent.TargetConf == 0 {
		c.Agent.TargetConf = 6
	}

	if c.Agent.MinChannelSize == 0 {
		c.Agent.MinChannelSize = 1_000_000
	}

	if c.Agent.MaxChannelSize == 0 {
		c.Agent.MaxChannelSize = 10_000_000
	}

	if c.Agent.ChannelManager.MinConf == 0 {
		c.Agent.ChannelManager.MinConf = 2
	}

	if c.Agent.ChannelManager.MaxSatvB == 0 {
		c.Agent.ChannelManager.MaxSatvB = 50
	}

	if c.Agent.ChannelManager.FeeRatePPM == 0 {
		c.Agent.ChannelManager.FeeRatePPM = 100
	}

	if c.Agent.HeuristicWeights.Open == (OpenWeights{}) {
		c.Agent.HeuristicWeights.Open = DefaultOpenWeights
	}

	if c.Agent.HeuristicWeights.Close == (CloseWeights{}) {
		c.Agent.HeuristicWeights.Close = DefaultCloseWeights
	}

	if c.Lightning.RPC.Timeout == 0 {
		c.Lightning.RPC.Timeout = 30 * time.Second
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
}

// Weights is a set of different heuristic weights.
type Weights interface {
	CloseWeights | OpenWeights | CentralityWeights | ChannelsWeights
}

// IterWeights iterates through the weights executing f on each of the values.
func IterWeights[T Weights](weights T, f func(weight float64) error) error {
	var err error
	w := reflect.ValueOf(weights)
	for i := range w.NumField() {
		weight := w.Field(i).Interface()
		switch weight.(type) {
		case float64:
			err = f(weight.(float64))
		case CentralityWeights:
			err = IterWeights(weight.(CentralityWeights), func(v float64) error {
				return f(v)
			})
		case ChannelsWeights:
			err = IterWeights(weight.(ChannelsWeights), func(v float64) error {
				return f(v)
			})
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// SumWeights returns the sum of the values stored in the weights objects.
func SumWeights[T Weights](weights T) float64 {
	sum := 0.0
	IterWeights(weights, func(weight float64) error {
		sum += weight
		return nil
	})

	return sum
}
