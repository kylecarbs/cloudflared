package ingress

import (
	"encoding/json"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/cloudflare/cloudflared/config"
	"github.com/cloudflare/cloudflared/ipaccess"
	"github.com/cloudflare/cloudflared/tlsconfig"
)

const (
	defaultConnectTimeout       = 30 * time.Second
	defaultTLSTimeout           = 10 * time.Second
	defaultTCPKeepAlive         = 30 * time.Second
	defaultKeepAliveConnections = 100
	defaultKeepAliveTimeout     = 90 * time.Second
	defaultProxyAddress         = "127.0.0.1"

	SSHServerFlag                 = "ssh-server"
	Socks5Flag                    = "socks5"
	ProxyConnectTimeoutFlag       = "proxy-connect-timeout"
	ProxyTLSTimeoutFlag           = "proxy-tls-timeout"
	ProxyTCPKeepAliveFlag         = "proxy-tcp-keepalive"
	ProxyNoHappyEyeballsFlag      = "proxy-no-happy-eyeballs"
	ProxyKeepAliveConnectionsFlag = "proxy-keepalive-connections"
	ProxyKeepAliveTimeoutFlag     = "proxy-keepalive-timeout"
	HTTPHostHeaderFlag            = "http-host-header"
	OriginServerNameFlag          = "origin-server-name"
	NoTLSVerifyFlag               = "no-tls-verify"
	NoChunkedEncodingFlag         = "no-chunked-encoding"
	ProxyAddressFlag              = "proxy-address"
	ProxyPortFlag                 = "proxy-port"
)

const (
	socksProxy = "socks"
)

// RemoteConfig models ingress settings that can be managed remotely, for example through the dashboard.
type RemoteConfig struct {
	Ingress     Ingress
	WarpRouting config.WarpRoutingConfig
}

type remoteConfigJSON struct {
	GlobalOriginRequest config.OriginRequestConfig      `json:"originRequest"`
	IngressRules        []config.UnvalidatedIngressRule `json:"ingress"`
	WarpRouting         config.WarpRoutingConfig        `json:"warp-routing"`
}

func (rc *RemoteConfig) UnmarshalJSON(b []byte) error {
	var rawConfig remoteConfigJSON
	if err := json.Unmarshal(b, &rawConfig); err != nil {
		return err
	}
	ingress, err := validateIngress(rawConfig.IngressRules, originRequestFromConfig(rawConfig.GlobalOriginRequest))
	if err != nil {
		return err
	}

	rc.Ingress = ingress
	rc.WarpRouting = rawConfig.WarpRouting

	return nil
}

func originRequestFromSingeRule(c *cli.Context) OriginRequestConfig {
	var connectTimeout time.Duration = defaultConnectTimeout
	var tlsTimeout time.Duration = defaultTLSTimeout
	var tcpKeepAlive time.Duration = defaultTCPKeepAlive
	var noHappyEyeballs bool
	var keepAliveConnections int = defaultKeepAliveConnections
	var keepAliveTimeout time.Duration = defaultKeepAliveTimeout
	var httpHostHeader string
	var originServerName string
	var caPool string
	var noTLSVerify bool
	var disableChunkedEncoding bool
	var bastionMode bool
	var proxyAddress = defaultProxyAddress
	var proxyPort uint
	var proxyType string
	if flag := ProxyConnectTimeoutFlag; c.IsSet(flag) {
		connectTimeout = c.Duration(flag)
	}
	if flag := ProxyTLSTimeoutFlag; c.IsSet(flag) {
		tlsTimeout = c.Duration(flag)
	}
	if flag := ProxyTCPKeepAliveFlag; c.IsSet(flag) {
		tcpKeepAlive = c.Duration(flag)
	}
	if flag := ProxyNoHappyEyeballsFlag; c.IsSet(flag) {
		noHappyEyeballs = c.Bool(flag)
	}
	if flag := ProxyKeepAliveConnectionsFlag; c.IsSet(flag) {
		keepAliveConnections = c.Int(flag)
	}
	if flag := ProxyKeepAliveTimeoutFlag; c.IsSet(flag) {
		keepAliveTimeout = c.Duration(flag)
	}
	if flag := HTTPHostHeaderFlag; c.IsSet(flag) {
		httpHostHeader = c.String(flag)
	}
	if flag := OriginServerNameFlag; c.IsSet(flag) {
		originServerName = c.String(flag)
	}
	if flag := tlsconfig.OriginCAPoolFlag; c.IsSet(flag) {
		caPool = c.String(flag)
	}
	if flag := NoTLSVerifyFlag; c.IsSet(flag) {
		noTLSVerify = c.Bool(flag)
	}
	if flag := NoChunkedEncodingFlag; c.IsSet(flag) {
		disableChunkedEncoding = c.Bool(flag)
	}
	if flag := config.BastionFlag; c.IsSet(flag) {
		bastionMode = c.Bool(flag)
	}
	if flag := ProxyAddressFlag; c.IsSet(flag) {
		proxyAddress = c.String(flag)
	}
	if flag := ProxyPortFlag; c.IsSet(flag) {
		// Note TUN-3758 , we use Int because UInt is not supported with altsrc
		proxyPort = uint(c.Int(flag))
	}
	if c.IsSet(Socks5Flag) {
		proxyType = socksProxy
	}
	return OriginRequestConfig{
		ConnectTimeout:         connectTimeout,
		TLSTimeout:             tlsTimeout,
		TCPKeepAlive:           tcpKeepAlive,
		NoHappyEyeballs:        noHappyEyeballs,
		KeepAliveConnections:   keepAliveConnections,
		KeepAliveTimeout:       keepAliveTimeout,
		HTTPHostHeader:         httpHostHeader,
		OriginServerName:       originServerName,
		CAPool:                 caPool,
		NoTLSVerify:            noTLSVerify,
		DisableChunkedEncoding: disableChunkedEncoding,
		BastionMode:            bastionMode,
		ProxyAddress:           proxyAddress,
		ProxyPort:              proxyPort,
		ProxyType:              proxyType,
	}
}

func originRequestFromConfig(c config.OriginRequestConfig) OriginRequestConfig {
	out := OriginRequestConfig{
		ConnectTimeout:       defaultConnectTimeout,
		TLSTimeout:           defaultTLSTimeout,
		TCPKeepAlive:         defaultTCPKeepAlive,
		KeepAliveConnections: defaultKeepAliveConnections,
		KeepAliveTimeout:     defaultKeepAliveTimeout,
		ProxyAddress:         defaultProxyAddress,
	}
	if c.ConnectTimeout != nil {
		out.ConnectTimeout = c.ConnectTimeout.Duration
	}
	if c.TLSTimeout != nil {
		out.TLSTimeout = c.TLSTimeout.Duration
	}
	if c.TCPKeepAlive != nil {
		out.TCPKeepAlive = c.TCPKeepAlive.Duration
	}
	if c.NoHappyEyeballs != nil {
		out.NoHappyEyeballs = *c.NoHappyEyeballs
	}
	if c.KeepAliveConnections != nil {
		out.KeepAliveConnections = *c.KeepAliveConnections
	}
	if c.KeepAliveTimeout != nil {
		out.KeepAliveTimeout = c.KeepAliveTimeout.Duration
	}
	if c.HTTPHostHeader != nil {
		out.HTTPHostHeader = *c.HTTPHostHeader
	}
	if c.OriginServerName != nil {
		out.OriginServerName = *c.OriginServerName
	}
	if c.CAPool != nil {
		out.CAPool = *c.CAPool
	}
	if c.NoTLSVerify != nil {
		out.NoTLSVerify = *c.NoTLSVerify
	}
	if c.DisableChunkedEncoding != nil {
		out.DisableChunkedEncoding = *c.DisableChunkedEncoding
	}
	if c.BastionMode != nil {
		out.BastionMode = *c.BastionMode
	}
	if c.ProxyAddress != nil {
		out.ProxyAddress = *c.ProxyAddress
	}
	if c.ProxyPort != nil {
		out.ProxyPort = *c.ProxyPort
	}
	if c.ProxyType != nil {
		out.ProxyType = *c.ProxyType
	}
	if len(c.IPRules) > 0 {
		for _, r := range c.IPRules {
			rule, err := ipaccess.NewRuleByCIDR(r.Prefix, r.Ports, r.Allow)
			if err == nil {
				out.IPRules = append(out.IPRules, rule)
			}
		}
	}
	return out
}

// OriginRequestConfig configures how Cloudflared sends requests to origin
// services.
// Note: To specify a time.Duration in go-yaml, use e.g. "3s" or "24h".
type OriginRequestConfig struct {
	// HTTP proxy timeout for establishing a new connection
	ConnectTimeout time.Duration `yaml:"connectTimeout"`
	// HTTP proxy timeout for completing a TLS handshake
	TLSTimeout time.Duration `yaml:"tlsTimeout"`
	// HTTP proxy TCP keepalive duration
	TCPKeepAlive time.Duration `yaml:"tcpKeepAlive"`
	// HTTP proxy should disable "happy eyeballs" for IPv4/v6 fallback
	NoHappyEyeballs bool `yaml:"noHappyEyeballs"`
	// HTTP proxy timeout for closing an idle connection
	KeepAliveTimeout time.Duration `yaml:"keepAliveTimeout"`
	// HTTP proxy maximum keepalive connection pool size
	KeepAliveConnections int `yaml:"keepAliveConnections"`
	// Sets the HTTP Host header for the local webserver.
	HTTPHostHeader string `yaml:"httpHostHeader"`
	// Hostname on the origin server certificate.
	OriginServerName string `yaml:"originServerName"`
	// Path to the CA for the certificate of your origin.
	// This option should be used only if your certificate is not signed by Cloudflare.
	CAPool string `yaml:"caPool"`
	// Disables TLS verification of the certificate presented by your origin.
	// Will allow any certificate from the origin to be accepted.
	// Note: The connection from your machine to Cloudflare's Edge is still encrypted.
	NoTLSVerify bool `yaml:"noTLSVerify"`
	// Disables chunked transfer encoding.
	// Useful if you are running a WSGI server.
	DisableChunkedEncoding bool `yaml:"disableChunkedEncoding"`
	// Runs as jump host
	BastionMode bool `yaml:"bastionMode"`
	// Listen address for the proxy.
	ProxyAddress string `yaml:"proxyAddress"`
	// Listen port for the proxy.
	ProxyPort uint `yaml:"proxyPort"`
	// What sort of proxy should be started
	ProxyType string `yaml:"proxyType"`
	// IP rules for the proxy service
	IPRules []ipaccess.Rule `yaml:"ipRules"`
}

func (defaults *OriginRequestConfig) setConnectTimeout(overrides config.OriginRequestConfig) {
	if val := overrides.ConnectTimeout; val != nil {
		defaults.ConnectTimeout = val.Duration
	}
}

func (defaults *OriginRequestConfig) setTLSTimeout(overrides config.OriginRequestConfig) {
	if val := overrides.TLSTimeout; val != nil {
		defaults.TLSTimeout = val.Duration
	}
}

func (defaults *OriginRequestConfig) setNoHappyEyeballs(overrides config.OriginRequestConfig) {
	if val := overrides.NoHappyEyeballs; val != nil {
		defaults.NoHappyEyeballs = *val
	}
}

func (defaults *OriginRequestConfig) setKeepAliveConnections(overrides config.OriginRequestConfig) {
	if val := overrides.KeepAliveConnections; val != nil {
		defaults.KeepAliveConnections = *val
	}
}

func (defaults *OriginRequestConfig) setKeepAliveTimeout(overrides config.OriginRequestConfig) {
	if val := overrides.KeepAliveTimeout; val != nil {
		defaults.KeepAliveTimeout = val.Duration
	}
}

func (defaults *OriginRequestConfig) setTCPKeepAlive(overrides config.OriginRequestConfig) {
	if val := overrides.TCPKeepAlive; val != nil {
		defaults.TCPKeepAlive = val.Duration
	}
}

func (defaults *OriginRequestConfig) setHTTPHostHeader(overrides config.OriginRequestConfig) {
	if val := overrides.HTTPHostHeader; val != nil {
		defaults.HTTPHostHeader = *val
	}
}

func (defaults *OriginRequestConfig) setOriginServerName(overrides config.OriginRequestConfig) {
	if val := overrides.OriginServerName; val != nil {
		defaults.OriginServerName = *val
	}
}

func (defaults *OriginRequestConfig) setCAPool(overrides config.OriginRequestConfig) {
	if val := overrides.CAPool; val != nil {
		defaults.CAPool = *val
	}
}

func (defaults *OriginRequestConfig) setNoTLSVerify(overrides config.OriginRequestConfig) {
	if val := overrides.NoTLSVerify; val != nil {
		defaults.NoTLSVerify = *val
	}
}

func (defaults *OriginRequestConfig) setDisableChunkedEncoding(overrides config.OriginRequestConfig) {
	if val := overrides.DisableChunkedEncoding; val != nil {
		defaults.DisableChunkedEncoding = *val
	}
}

func (defaults *OriginRequestConfig) setBastionMode(overrides config.OriginRequestConfig) {
	if val := overrides.BastionMode; val != nil {
		defaults.BastionMode = *val
	}
}

func (defaults *OriginRequestConfig) setProxyPort(overrides config.OriginRequestConfig) {
	if val := overrides.ProxyPort; val != nil {
		defaults.ProxyPort = *val
	}
}

func (defaults *OriginRequestConfig) setProxyAddress(overrides config.OriginRequestConfig) {
	if val := overrides.ProxyAddress; val != nil {
		defaults.ProxyAddress = *val
	}
}

func (defaults *OriginRequestConfig) setProxyType(overrides config.OriginRequestConfig) {
	if val := overrides.ProxyType; val != nil {
		defaults.ProxyType = *val
	}
}

func (defaults *OriginRequestConfig) setIPRules(overrides config.OriginRequestConfig) {
	if val := overrides.IPRules; len(val) > 0 {
		ipAccessRule := make([]ipaccess.Rule, len(overrides.IPRules))
		for i, r := range overrides.IPRules {
			rule, err := ipaccess.NewRuleByCIDR(r.Prefix, r.Ports, r.Allow)
			if err == nil {
				ipAccessRule[i] = rule
			}
		}
		defaults.IPRules = ipAccessRule
	}
}

// SetConfig gets config for the requests that cloudflared sends to origins.
// Each field has a setter method which sets a value for the field by trying to find:
//   1. The user config for this rule
//   2. The user config for the overall ingress config
//   3. Defaults chosen by the cloudflared team
//   4. Golang zero values for that type
// If an earlier option isn't set, it will try the next option down.
func setConfig(defaults OriginRequestConfig, overrides config.OriginRequestConfig) OriginRequestConfig {
	cfg := defaults
	cfg.setConnectTimeout(overrides)
	cfg.setTLSTimeout(overrides)
	cfg.setNoHappyEyeballs(overrides)
	cfg.setKeepAliveConnections(overrides)
	cfg.setKeepAliveTimeout(overrides)
	cfg.setTCPKeepAlive(overrides)
	cfg.setHTTPHostHeader(overrides)
	cfg.setOriginServerName(overrides)
	cfg.setCAPool(overrides)
	cfg.setNoTLSVerify(overrides)
	cfg.setDisableChunkedEncoding(overrides)
	cfg.setBastionMode(overrides)
	cfg.setProxyPort(overrides)
	cfg.setProxyAddress(overrides)
	cfg.setProxyType(overrides)
	cfg.setIPRules(overrides)
	return cfg
}
