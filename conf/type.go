package conf

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"
	"time"

	aresv1 "ares-operator/api/v1"

	"gopkg.in/yaml.v3"
)

/*****************
 * Configuration
 *****************/
type Configuration struct {
	Operator     OperatorConfig                `yaml:"operator,omitempty"`
	Cache        CacheConfig                   `yaml:"cache,omitempty"`
	Initializers map[string]*InitializerConfig `yaml:"initializers,omitempty"`
}

func (c *Configuration) Load(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(content, c)
	if err != nil {
		return err
	}
	return c.Validate()
}

func (c *Configuration) Validate() error {
	if err := c.Cache.Validate("$.cache"); err != nil {
		return err
	}
	if err := c.Operator.Validate("$.operator"); err != nil {
		return err
	}
	for t, initConfig := range c.Initializers {
		if err := initConfig.Validate(fmt.Sprintf("$.%s", t)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Configuration) GetCommonConfig() (*InitializerConfig, error) {
	config, ok := c.Initializers["common"]
	if !ok {
		return nil, errors.New("common initializer config is not found")
	}
	return config, nil
}

func (c *Configuration) TTLSecondsAfterSucceeded() *time.Duration {
	seconds := c.Operator.AresJob.TTLSecondsAfterSucceeded
	if seconds == nil {
		return nil
	}
	ttl := time.Duration(*seconds) * time.Second
	return &ttl
}

func (c *Configuration) GetRestConfig() RestConfig {
	return c.Operator.Rest
}

/*****************
 * OperatorConfig
 *****************/
type OperatorConfig struct {
	Rest    RestConfig    `yaml:"restConfig,omitempty"`
	AresJob AresJobConfig `yaml:"aresjob,omitempty"`
}

func (c *OperatorConfig) Validate(prefix string) error {
	if err := c.Rest.Validate(fmt.Sprintf("%s.restConfig", prefix)); err != nil {
		return err
	}
	return nil
}

type AresJobConfig struct {
	TTLSecondsAfterSucceeded *int32 `yaml:"ttlSecondsAfterSucceeded,omitempty"`
}

/*********
 * Cache
 *********/
type CacheConfig struct {
	URI      string         `yaml:"uri,omitempty"`
	Observer ObserverConfig `yaml:"observer,omitempty"`
}

func (c *CacheConfig) Validate(prefix string) error {
	if err := c.Observer.Validate(fmt.Sprintf("%s.observer", prefix)); err != nil {
		return err
	}
	return nil
}

func (c *CacheConfig) NodePortEnabled() bool {
	return c.Observer.Enabled && c.Observer.NodePortEnabled()
}

func (c *CacheConfig) GetURI() string {
	if c.Observer.Enabled {
		return c.Observer.URI()
	}
	return c.URI
}

type ObserverConfig struct {
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	Service   string            `json:"service" yaml:"service"`
	NodePort  int               `json:"nodePort" yaml:"nodePort"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Labels    map[string]string `json:"labels" yaml:"labels"`
}

func (c *ObserverConfig) Validate(prefix string) error {
	if c.NodePortEnabled() {
		if len(c.Namespace) == 0 {
			return fmt.Errorf("%s.namespace is required", prefix)
		}
		if len(c.Labels) == 0 {
			return fmt.Errorf("%s.labels is required", prefix)
		}
	}
	return nil
}

func (c *ObserverConfig) NodePortEnabled() bool {
	return c.NodePort > 0
}

func (c *ObserverConfig) NodePortURI(ips ...string) string {
	ip := strings.Join(ips, ",")
	if len(ip) > 0 {
		ip = fmt.Sprintf("%s:%d", ip, c.NodePort)
	} else {
		ip = fmt.Sprintf("%d", c.NodePort)
	}
	return fmt.Sprintf("nodePort://%s", ip)
}

func (c *ObserverConfig) URI() string {
	if c.NodePortEnabled() {
		return c.NodePortURI()
	}
	return fmt.Sprintf("http://%s", c.Service)
}

/*************
 * RestConfig
 *************/
type RestConfig struct {
	QPS   float32 `json:"qps" yaml:"qps"`
	Burst int     `json:"burst" yaml:"burst"`
}

func (c *RestConfig) Validate(prefix string) error {
	if c.QPS <= 0 || c.Burst <= 0 {
		return fmt.Errorf("%s.qps and %s.burst must be positive", prefix, prefix)
	}
	return nil
}

/**************
 * Initializer
 **************/
type InitializerConfig struct {
	Image       string             `yaml:"image,omitempty"`
	Cmd         string             `yaml:"cmd,omitempty"`
	CmdTemplate *template.Template `yaml:"-"`
	Envs        map[string]string  `yaml:"envs,omitempty"`
	Volumes     []VolumeConfig     `yaml:"volumes,omitempty"`
}

type VolumeConfig struct {
	Name     string `yaml:"name,omitempty"`
	HostPath string `yaml:"hostPath,omitempty"`
}

type CmdConfig struct {
	Name              string
	Namespace         string
	CacheURI          string
	Key               string
	Roles             string
	MPIHostFile       string
	ReplicasPerNode   int
	MPIImplementation aresv1.MPIImplementation
}

func (c *InitializerConfig) Validate(prefix string) error {
	if len(c.Image) == 0 {
		return fmt.Errorf("%s.image is required", prefix)
	}
	if t, err := template.New("cmdTemplate").Parse(c.Cmd); err != nil {
		return fmt.Errorf("%s.cmd parse failed: %v", prefix, err)
	} else {
		c.CmdTemplate = t
	}
	for i, volume := range c.Volumes {
		if len(volume.Name) == 0 || len(volume.HostPath) == 0 {
			return fmt.Errorf("%s.volumes[%d] is not completed", prefix, i)
		}
	}
	return nil
}
