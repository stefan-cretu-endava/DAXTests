/*
  Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.

  Licensed under the Apache License, Version 2.0 (the "License").
  You may not use this file except in compliance with the License.
  A copy of the License is located at

      http://www.apache.org/licenses/LICENSE-2.0

  or in the "license" file accompanying this file. This file is distributed
  on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
  express or implied. See the License for the specific language governing
  permissions and limitations under the License.
*/

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-dax-go-v2/dax/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/logging"
)

type serviceEndpoint struct {
	nodeId           int64
	hostname         string
	address          []byte
	port             int
	role             int
	availabilityZone string
	leaderSessionId  int64
}

func (e *serviceEndpoint) hostPort() hostPort {
	return hostPort{net.IP(e.address).String(), e.port}
}

type hostPort struct {
	host string
	port int
}

type Config struct {
	MaxPendingConnectionsPerHost int
	ClusterUpdateThreshold       time.Duration
	ClusterUpdateInterval        time.Duration
	IdleConnectionReapDelay      time.Duration
	ClientHealthCheckInterval    time.Duration

	Region      string
	HostPorts   []string
	Credentials aws.CredentialsProvider
	DialContext func(ctx context.Context, network string, address string) (net.Conn, error)
	connConfig  connConfig

	SkipHostnameVerification bool
	logger                   logging.Logger
	logLevel                 utils.LogLevelType

	RouteManagerEnabled bool // this flag temporarily removes routes facing network errors.
	IpDiscovery         string
}

type connConfig struct {
	isEncrypted              bool
	hostname                 string
	skipHostnameVerification bool
}

func (cfg *Config) validate() error {
	if cfg.HostPorts == nil || len(cfg.HostPorts) == 0 {
		return smithy.NewErrParamRequired("Endpoint")
	}
	if len(cfg.Region) == 0 {
		return smithy.NewErrParamRequired("config.Region")
	}
	if cfg.Credentials == nil {
		return smithy.NewErrParamRequired("config.Credentials")
	}
	if cfg.MaxPendingConnectionsPerHost < 0 {
		return NewCustomInvalidParamError("ConfigValidation", "MaxPendingConnectionsPerHost cannot be negative")
	}
	return nil
}

func (cfg *Config) validateConnConfig() {
	if cfg.connConfig.isEncrypted && cfg.SkipHostnameVerification {
		cfg.logger.Logf(logging.Warn, "Skip hostname verification of TLS connections. The default is to perform hostname verification, setting this to True will skip verification. Be sure you understand the implication of doing so, which is the inability to authenticate the cluster that you are connecting to.")
	}
}

func (cfg *Config) SetLogger(logger logging.Logger, logLevel utils.LogLevelType) {
	cfg.logger = logger
	cfg.logLevel = logLevel
}

var defaultPorts = map[string]int{
	"dax":  8111,
	"daxs": 9111,
}

func DefaultConfig() Config {
	cfg := Config{
		MaxPendingConnectionsPerHost: 10,
		ClusterUpdateInterval:        time.Second * 4,
		ClusterUpdateThreshold:       time.Millisecond * 125,
		ClientHealthCheckInterval:    time.Second * 5,

		connConfig:               connConfig{},
		SkipHostnameVerification: false,
		logger:                   utils.NewDefaultLogger(),
		logLevel:                 utils.LogOff,
		IdleConnectionReapDelay:  30 * time.Second,
		RouteManagerEnabled:      false,
		IpDiscovery:              "",
	}
	if cfg.Credentials == nil {
		conf, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			panic(fmt.Sprintf("unexpected error: %+v", err))
		}
		cfg.Credentials = conf.Credentials
	}
	return cfg
}

type ClusterDaxClient struct {
	config  Config
	cluster *cluster
}

func New(config Config) (*ClusterDaxClient, error) {
	cluster, err := newCluster(config)
	if err != nil {
		return nil, err
	}
	err = cluster.start()
	if err != nil {
		return nil, err
	}
	client := &ClusterDaxClient{config: config, cluster: cluster}
	return client, nil
}

func (cc *ClusterDaxClient) Close() error {
	return cc.cluster.Close()
}

func (cc *ClusterDaxClient) endpoints(ctx context.Context, opt RequestOptions) ([]serviceEndpoint, error) {
	var out []serviceEndpoint
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		out, err = client.endpoints(ctx, o)
		return err
	}
	if err = cc.retry(ctx, opEndpoints, action, opt); err != nil {
		return nil, err
	}
	return out, nil
}

func (cc *ClusterDaxClient) PutItemWithOptions(ctx context.Context, input *dynamodb.PutItemInput, output *dynamodb.PutItemOutput, opt RequestOptions) (*dynamodb.PutItemOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.PutItemWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpPutItem, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) DeleteItemWithOptions(ctx context.Context, input *dynamodb.DeleteItemInput, output *dynamodb.DeleteItemOutput, opt RequestOptions) (*dynamodb.DeleteItemOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.DeleteItemWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpDeleteItem, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) UpdateItemWithOptions(ctx context.Context, input *dynamodb.UpdateItemInput, output *dynamodb.UpdateItemOutput, opt RequestOptions) (*dynamodb.UpdateItemOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.UpdateItemWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpUpdateItem, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) BatchWriteItemWithOptions(ctx context.Context, input *dynamodb.BatchWriteItemInput, output *dynamodb.BatchWriteItemOutput, opt RequestOptions) (*dynamodb.BatchWriteItemOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.BatchWriteItemWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpBatchWriteItem, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) TransactWriteItemsWithOptions(ctx context.Context, input *dynamodb.TransactWriteItemsInput, output *dynamodb.TransactWriteItemsOutput, opt RequestOptions) (*dynamodb.TransactWriteItemsOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.TransactWriteItemsWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpTransactWriteItems, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) TransactGetItemsWithOptions(ctx context.Context, input *dynamodb.TransactGetItemsInput, output *dynamodb.TransactGetItemsOutput, opt RequestOptions) (*dynamodb.TransactGetItemsOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.TransactGetItemsWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpTransactGetItems, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) GetItemWithOptions(ctx context.Context, input *dynamodb.GetItemInput, output *dynamodb.GetItemOutput, opt RequestOptions) (*dynamodb.GetItemOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.GetItemWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpGetItem, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) QueryWithOptions(ctx context.Context, input *dynamodb.QueryInput, output *dynamodb.QueryOutput, opt RequestOptions) (*dynamodb.QueryOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.QueryWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpQuery, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) ScanWithOptions(ctx context.Context, input *dynamodb.ScanInput, output *dynamodb.ScanOutput, opt RequestOptions) (*dynamodb.ScanOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.ScanWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpScan, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) BatchGetItemWithOptions(ctx context.Context, input *dynamodb.BatchGetItemInput, output *dynamodb.BatchGetItemOutput, opt RequestOptions) (*dynamodb.BatchGetItemOutput, error) {
	var err error
	action := func(client DaxAPI, o RequestOptions) error {
		output, err = client.BatchGetItemWithOptions(ctx, input, output, o)
		return err
	}
	if err = cc.retry(ctx, OpBatchGetItem, action, opt); err != nil {
		return output, err
	}
	return output, nil
}

func (cc *ClusterDaxClient) retry(ctx context.Context, op string, action func(client DaxAPI, o RequestOptions) error, opt RequestOptions) (err error) {
	defer func() {
		if daxErr, ok := err.(daxError); ok {
			err = convertDaxError(daxErr)
		}
	}()

	ctx = cc.newContext(ctx, opt)

	attempts := opt.RetryMaxAttempts
	opt.RetryMaxAttempts = 0 // disable retries on single node client

	var client DaxAPI
	// Start from 0 to accomodate for the initial request
	for i := 0; i <= attempts; i++ {
		if i > 0 && opt.Logger != nil && opt.LogLevel.Matches(utils.LogDebugWithRequestRetries) {
			opt.Logger.Logf(logging.Debug, "Retrying Request %s/%s, attempt %d", service, op, i)
		}
		client, err = cc.cluster.client(client, op)

		if err == nil {
			err = action(client, opt)
		}

		if err == nil {
			// success
			return nil
		}
		if !isRetryable(opt, err) {
			return err
		}

		if i != attempts {
			if opt.Logger != nil && opt.LogLevel.Matches(utils.LogDebugWithRequestRetries) {
				opt.Logger.Logf(logging.Debug, "Error in executing request %s/%s. : %s", service, op, err)
			}

			var delay time.Duration
			delay = opt.Retryer.RetryDelay(i+1, err)
			if delay == 0 {
				delay = opt.RetryDelay
			}

			if delay > 0 {
				if err = SleepWithContext(ctx, op, delay); err != nil {
					return err
				}
			}
		}
	}
	return err
}

func (cc *ClusterDaxClient) newContext(ctx context.Context, o RequestOptions) context.Context {
	if o.Context != nil {
		return o.Context
	}
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

type cluster struct {
	lock           sync.RWMutex
	active         map[hostPort]clientAndConfig // protected by lock
	routeManager   RouteManager                 // protected by lock
	closed         bool                         // protected by lock
	lastRefreshErr error                        // protected by lock

	lastUpdateNs int64
	executor     *taskExecutor

	seeds         []hostPort
	config        Config
	clientBuilder clientBuilder
	IpDiscovery   string
}

type clientAndConfig struct {
	client DaxAPI
	cfg    serviceEndpoint
}

func newCluster(cfg Config) (*cluster, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	seeds, hostname, isEncrypted, err := getHostPorts(cfg.HostPorts)
	if err != nil {
		return nil, err
	}
	cfg.connConfig.isEncrypted = isEncrypted
	cfg.connConfig.skipHostnameVerification = cfg.SkipHostnameVerification
	cfg.connConfig.hostname = hostname
	cfg.validateConnConfig()
	routeManager := newRouteManager(cfg.RouteManagerEnabled, cfg.ClientHealthCheckInterval, cfg.logger, cfg.logLevel)
	return &cluster{seeds: seeds, config: cfg, executor: newExecutor(), clientBuilder: &singleClientBuilder{}, routeManager: routeManager, IpDiscovery: cfg.IpDiscovery}, nil
}

func getHostPorts(hosts []string) (hostPorts []hostPort, hostname string, isEncrypted bool, err error) {
	out := make([]hostPort, len(hosts))

	handle := func(e error) (hostPorts []hostPort, hostname string, isEncrypted bool, err error) {
		return nil, "", false, e
	}

	for i, hp := range hosts {
		host, port, scheme, err := parseHostPort(hp)
		if err != nil {
			return handle(err)
		}

		if isEncrypted != (scheme == "daxs") {
			if i == 0 {
				isEncrypted = true
			} else {
				return handle(&smithy.GenericAPIError{
					Code:    ErrCodeInvalidParameter,
					Message: fmt.Sprintf("Inconsistency between the schemes of provided endpoints"),
					Fault:   smithy.FaultClient,
				})
			}
		}
		if scheme == "daxs" && i > 0 {
			return handle(&smithy.GenericAPIError{
				Code:    ErrCodeInvalidParameter,
				Message: fmt.Sprintf("Only one cluster discovery endpoint may be provided for encrypted cluster"),
				Fault:   smithy.FaultClient,
			})
		}
		out[i] = hostPort{host, port}
		hostname = host
	}
	return out, hostname, isEncrypted, nil
}

func parseHostPort(hostPort string) (host string, port int, scheme string, err error) {
	uriString := hostPort
	colon := strings.Index(hostPort, "://")

	handle := func(e error) (host string, port int, scheme string, err error) {
		return "", 0, "", e
	}

	if colon == -1 {
		if strings.Index(hostPort, ":") == -1 {
			return handle(&smithy.GenericAPIError{
				Code:    ErrCodeInvalidParameter,
				Message: fmt.Sprintf(hostPort + "Invalid hostport."),
				Fault:   smithy.FaultClient,
			})
		}
		uriString = "dax://" + hostPort
	}
	u, err := url.ParseRequestURI(uriString)
	if err != nil {
		return handle(err)
	}

	host = u.Hostname()
	scheme = u.Scheme
	portStr := u.Port()
	if host == "" {
		return handle(&smithy.GenericAPIError{
			Code:    ErrCodeInvalidParameter,
			Message: fmt.Sprintf("Invalid hostport"),
			Fault:   smithy.FaultClient,
		})
	}

	port, err = strconv.Atoi(portStr)
	if err != nil {
		port = defaultPorts[scheme]
	}

	if _, ok := defaultPorts[scheme]; !ok {
		schemes := strings.Join(strings.Fields(fmt.Sprint(reflect.ValueOf(defaultPorts).MapKeys())), ",")
		return handle(&smithy.GenericAPIError{
			Code:    ErrCodeInvalidParameter,
			Message: fmt.Sprintf("URL scheme must be one of " + schemes),
			Fault:   smithy.FaultClient,
		})
	}

	return host, port, scheme, nil
}

func (c *cluster) start() error {
	c.executor.start(c.config.ClusterUpdateInterval, func() error {
		c.safeRefresh(false)
		return nil
	})
	c.executor.start(c.config.IdleConnectionReapDelay, c.reapIdleConnections)
	c.safeRefresh(false)
	return nil
}

func (c *cluster) Close() error {
	c.executor.stopAll()

	c.lock.Lock()
	defer c.lock.Unlock()
	c.closed = true
	for _, config := range c.active {
		c.closeClient(config.client)
	}
	c.active = nil
	c.routeManager.close()
	c.routeManager = nil
	return nil
}

func (c *cluster) reapIdleConnections() error {
	clients := c.getAllRoutes()
	for _, c := range clients {
		if d, ok := c.(connectionReaper); ok {
			d.reapIdleConnections()
		}
	}
	return nil
}

func (c *cluster) client(prev DaxAPI, op string) (DaxAPI, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	route := c.routeManager.getRoute(prev)
	if route == nil {
		return nil, &smithy.OperationError{
			ServiceID:     service,
			OperationName: op,
			Err:           fmt.Errorf("no routes found. lastRefreshError: %v", c.lastRefreshError()),
		}
	}
	return route, nil
}

func (c *cluster) safeRefresh(force bool) {
	err := c.refresh(force)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.lastRefreshErr = err
}

func (c *cluster) lastRefreshError() error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lastRefreshErr
}

func (c *cluster) refresh(force bool) error {
	last := atomic.LoadInt64(&c.lastUpdateNs)
	now := time.Now().UnixNano()
	if now-last > c.config.ClusterUpdateThreshold.Nanoseconds() || force {
		if atomic.CompareAndSwapInt64(&c.lastUpdateNs, last, now) {
			return c.refreshNow()
		}
	}
	return nil
}

func (c *cluster) refreshNow() error {
	cfg, err := c.pullEndpoints()
	if err != nil {
		c.debugLog("ERROR: Failed to refresh endpoint : %s", err)
		return err
	}
	if !c.hasChanged(cfg) {
		return nil
	}
	return c.update(cfg)
}

// This method is responsible for updating the set of active routes tracked by
// the clsuter-dax-client in response to updates in the roster.
func (c *cluster) update(config []serviceEndpoint) error {
	newEndpoints := make(map[hostPort]struct{}, len(config))
	for _, cfg := range config {
		newEndpoints[cfg.hostPort()] = struct{}{}
	}

	newActive := make(map[hostPort]clientAndConfig, len(config))
	newRoutes := make([]DaxAPI, len(config))
	shouldUpdateRoutes := true
	var toClose []clientAndConfig
	// Track the newly created client instances, so that we can clean them up in case of partial failures.
	var newCliCfg []clientAndConfig

	c.lock.Lock()

	cls := c.closed
	oldActive := c.active

	if cls {
		shouldUpdateRoutes = false
	} else {
		// Close the client instances that are no longer part of roster.
		for ep, clicfg := range oldActive {
			_, isPartOfUpdatedEndpointsConfig := newEndpoints[ep]
			if !isPartOfUpdatedEndpointsConfig {
				c.debugLog("Found updated endpoint configs, will close inactive endpoint client : %s", ep.host)
				toClose = append(toClose, clicfg)
			}
		}

		// Create client instances for the new endpoints in roster.
		for i, ep := range config {
			cliAndCfg, alreadyExists := oldActive[ep.hostPort()]
			if !alreadyExists {
				cli, err := c.newSingleClient(ep)
				if err != nil {
					shouldUpdateRoutes = false
					break
				} else {
					cliAndCfg = clientAndConfig{client: cli, cfg: ep}
					newCliCfg = append(newCliCfg, cliAndCfg)
				}

				if singleCli, ok := cli.(HealthCheckDaxAPI); ok {
					singleCli.startHealthChecks(c, ep.hostPort())
				}
			}
			newActive[ep.hostPort()] = cliAndCfg
			newRoutes[i] = cliAndCfg.client
		}
	}

	if shouldUpdateRoutes {
		c.active = newActive
		c.routeManager.setRoutes(newRoutes)
	} else {
		// cleanup newly created clients if they are not going to be tracked further.
		toClose = append(toClose, newCliCfg...)
	}
	c.lock.Unlock()

	go func() {
		for _, client := range toClose {
			c.debugLog("Closing client for : %s", client.cfg.hostname)
			c.closeClient(client.client)
		}
	}()
	return nil
}

func (c *cluster) onHealthCheckFailed(host hostPort) {
	c.lock.Lock()
	c.debugLog("Refreshing cache for host: " + host.host)
	shouldCloseOldClient := true
	var oldClientConfig, ok = c.active[host]
	if ok {
		cli, err := c.newSingleClient(oldClientConfig.cfg)
		if singleCli, ok := cli.(HealthCheckDaxAPI); ok {
			singleCli.startHealthChecks(c, host)
		}

		if err == nil {
			c.active[host] = clientAndConfig{client: cli, cfg: oldClientConfig.cfg}

			newRoutes := make([]DaxAPI, len(c.active))
			i := 0
			for _, cliAndCfg := range c.active {
				newRoutes[i] = cliAndCfg.client
				i++
			}
			c.routeManager.setRoutes(newRoutes)
		} else {
			shouldCloseOldClient = false
			c.debugLog("Failed to refresh cache for host: " + host.host)
		}
	} else {
		c.debugLog("The node is not part of active routes. Ignoring the health check failure for host: " + host.host)
	}
	c.lock.Unlock()

	if shouldCloseOldClient {
		c.debugLog("Closing old instance of a replaced client for endpoint: %s", oldClientConfig.cfg.hostPort().host)
		c.closeClient(oldClientConfig.client)
	}
}

func (c *cluster) hasChanged(cfg []serviceEndpoint) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, se := range cfg {
		_, ok := c.active[se.hostPort()]
		if !ok {
			return true
		}
	}
	return len(cfg) != len(c.active)
}

/*
* Given a slice of IPs obtained after resolving a seed's address, separate it into 2 slices, one containing
* only IPv4 addresses, the other containing only IPv6 addresses. Later, both slices can be used to determine
* if the current seed supports dual stack (if both slices are not empty).
* Function not implemented on any struct, so it can be reused in more contexts.
* */
func SegregateIPsAssociatedWithSeed(ips []net.IP) (ipv4Addresses []net.IP, ipv6Addresses []net.IP) {
	for _, ip := range ips {
		// A seggregation based entirely on IP's length cannot be done, as IPv4 can use 16-byte representation, like IPv6
		if len(ip) == net.IPv4len || len(ip) == net.IPv6len {
			// To4() returns a non-nil value if the IP is IPv4. If nil is returned and IP isn't nil, then it is IPv6
			if ip.To4() != nil {
				ipv4Addresses = append(ipv4Addresses, ip)
			} else {
				ipv6Addresses = append(ipv6Addresses, ip)
			}
		}
	}

	return ipv4Addresses, ipv6Addresses
}

/*
* Check the configuration (IPv4/IPv6/Dual stack) of a given seed in cluster, based on the discovered IP addresses.
* Use the found configuration in conjunction with the user provided IpDiscovery to decide which ip set will be used
* to pull the endpoint from.
* Return a slice containing the appropriate IP addresses to be used, or an empty slice otherwise.
* */
func (c *cluster) handleDualStackScenarios(seedIPs []net.IP) (selectedIPs []net.IP) {

	c.debugLog("[handleDualStackScenarios] Segregating a IP list of current seed counting: " + strconv.Itoa(len(seedIPs)))
	// Seggregate IPv4 and IPv6 addresses of the current seed node, to determine if it is configured dual stack mode
	ipv4Addresses, ipv6Addresses := SegregateIPsAssociatedWithSeed(seedIPs)
	c.debugLog("[handleDualStackScenarios] Segregated IPv4 list size: " + strconv.Itoa(len(ipv4Addresses)) + " IPv6 list size: " + strconv.Itoa(len(ipv6Addresses)))

	// Determine the supported network type based on the discovered addresses
	isIPv4Seed := len(ipv4Addresses) > 0
	isIPv6Seed := len(ipv6Addresses) > 0

	/*
	* Implementation of IPv6 DAX feature involves determining the addresses type: IPv4, IPv6, dual stack.
	* This process always occurs after the resolution of endpoints' ip addresses.
	* Dual stack => an endpoint supports both IPv4 and IPv6, thus entails checking multiple IPs for determining it's usage
	* */
	isDualStackSeed := isIPv4Seed && isIPv6Seed
	shallIPv4BeUsed := false
	shallIPv6BeUsed := false

	// Check the optional IpDiscovery client-provided parameter which indicates the DAX client's IP type preference for connection
	if isDualStackSeed {
		if strings.EqualFold(c.IpDiscovery, "ipv4") || c.IpDiscovery == "" {
			// Use IPv4 connection if local DNS determined a dual stack config and the client opted for IPv4 or has no option
			shallIPv4BeUsed = true
			c.debugLog("[handleDualStackScenarios] dual stack configuration with IpDiscovery = " + c.IpDiscovery)
		} else if strings.EqualFold(c.IpDiscovery, "ipv6") {
			// Use IPv6 connection if local DNS determined a dual stack config and the client opted for IPv6
			shallIPv6BeUsed = true
			c.debugLog("[handleDualStackScenarios] dual stack configuration with ipv6 IpDiscovery = " + c.IpDiscovery)
		}
	} else if isIPv4Seed && !isIPv6Seed {
		if strings.EqualFold(c.IpDiscovery, "ipv4") || c.IpDiscovery == "" {
			// Use IPv4 connection if local DNS determined a IPv4 config and the client opted for IPv4 or has no option
			shallIPv4BeUsed = true
			c.debugLog("[handleDualStackScenarios] IPv4 configuration with IpDiscovery = " + c.IpDiscovery)
		} else if strings.EqualFold(c.IpDiscovery, "ipv6") {
			c.debugLog("IpDiscovery does not match the supported network type. IpDiscovery: ipv6, SupportedNetworkType: ipv4")
		}
	} else if !isIPv4Seed && isIPv6Seed {
		if strings.EqualFold(c.IpDiscovery, "ipv6") || c.IpDiscovery == "" {
			// Use IPv6 connection if local DNS determined a IPv6 config and the client opted for IPv6 or has no option
			shallIPv6BeUsed = true
			c.debugLog("[handleDualStackScenarios] IPv6 configuration with IpDiscovery = " + c.IpDiscovery)
		} else if strings.EqualFold(c.IpDiscovery, "ipv4") {
			c.debugLog("IpDiscovery does not match the supported network type. IpDiscovery: ipv4, SupportedNetworkType: ipv6")
		}
	}

	if shallIPv4BeUsed {
		c.debugLog("[handleDualStackScenarios] Use IPv4 ips based on the found config and provided IpDiscovery = " + c.IpDiscovery)
		return ipv4Addresses
	} else if shallIPv6BeUsed {
		c.debugLog("[handleDualStackScenarios] Use IPv6 ips based on the found config and provided IpDiscovery = " + c.IpDiscovery)
		return ipv6Addresses
	}

	c.debugLog("[handleDualStackScenarios] No endpoints are available based on the found config and provided IpDiscovery = " + c.IpDiscovery)
	// return empty list
	return
}

func (c *cluster) pullEndpoints() ([]serviceEndpoint, error) {
	var lastErr error // TODO chain errors?
	// Multiple seeds (known nodes with public address) are used as entry points for a given cluster, to handle fault tolerance
	for _, s := range c.seeds {
		// Address resolution: determine the IP addresses assigned to each known seed hostname.
		// A seed hostname can resolve to multiple IPs, both IPv4 and IPv6
		seedIPs, err := net.LookupIP(s.host)
		if err != nil {
			lastErr = err
			continue
		}

		if len(seedIPs) > 1 {
			// randomize multiple addresses; in-place fischer-yates shuffle.
			for idx := len(seedIPs) - 1; idx > 0; idx-- {
				shuffleIndex := rand.Intn(idx + 1)
				seedIPs[shuffleIndex], seedIPs[idx] = seedIPs[idx], seedIPs[shuffleIndex]
			}
		}

		c.debugLog("[pullEndpoints] Selecting available IPs for seed = " + s.host + strconv.Itoa(s.port))
		selectedIPsForCurrentSeed := c.handleDualStackScenarios(seedIPs)
		c.debugLog("[pullEndpoints] Available IPs for seed = " + s.host + strconv.Itoa(s.port) + " were selected")

		if len(selectedIPsForCurrentSeed) == 0 {
			c.debugLog("[pullEndpoints] NoRouteException: No endpoints available for seed = " + s.host + strconv.Itoa(s.port))
			lastErr = errors.New("NoRouteException: No endpoints available")
			//To determine if the code should continue from here or shall return like below ???

			return nil, &smithy.GenericAPIError{
				Code:    ErrCodeNoRouteException,
				Message: fmt.Sprintf("No endpoints available"),
				Fault:   smithy.FaultClient,
			}
		}

		for _, ip := range selectedIPsForCurrentSeed {
			endpoints, err := c.pullEndpointsFrom(ip, s.port)
			if err != nil {
				lastErr = err
				continue
			}
			c.debugLog("Pulled endpoints from %s : %v", ip, endpoints)
			if len(endpoints) > 0 {
				return endpoints, nil
			}
		}
	}
	return nil, lastErr
}

func (c *cluster) pullEndpointsFrom(ip net.IP, port int) ([]serviceEndpoint, error) {
	client, err := c.clientBuilder.newClient(ip, port, c.config.connConfig, c.config.Region, c.config.Credentials,
		c.config.MaxPendingConnectionsPerHost, c.config.DialContext, nil)
	if err != nil {
		return nil, err
	}
	defer c.closeClient(client)
	ctx, cfn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cfn()
	opts := RequestOptions{}
	opts.RetryMaxAttempts = 2
	return client.endpoints(ctx, opts)
}

func (c *cluster) closeClient(client DaxAPI) {
	if d, ok := client.(io.Closer); ok {
		d.Close()
	}
}

func (c *cluster) debugLog(logString string, args ...interface{}) {
	if c.config.logger != nil && c.config.logLevel.AtLeast(utils.LogDebug) {
		{
			c.config.logger.Logf(logging.Debug, logString, args...)
		}
	}
}

func (c *cluster) isRouteManagerEnabled() bool {
	return c.config.RouteManagerEnabled
}

func (c *cluster) addRoute(endpoint string, route DaxAPI) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.routeManager.addRoute(endpoint, route)
}

func (c *cluster) removeRoute(endpoint string, route DaxAPI) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.routeManager.removeRoute(endpoint, route, c.active)
}

func (c *cluster) getAllRoutes() []DaxAPI {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.routeManager.getAllRoutes()
}

func (c *cluster) newSingleClient(cfg serviceEndpoint) (DaxAPI, error) {
	return c.clientBuilder.newClient(net.IP(cfg.address), cfg.port, c.config.connConfig, c.config.Region, c.config.Credentials, c.config.MaxPendingConnectionsPerHost, c.config.DialContext, c)
}

type clientBuilder interface {
	newClient(net.IP, int, connConfig, string, aws.CredentialsProvider, int, dialContext, RouteListener) (DaxAPI, error)
}

type singleClientBuilder struct{}

func (*singleClientBuilder) newClient(ip net.IP, port int, connConfigData connConfig, region string, credentials aws.CredentialsProvider, maxPendingConnects int, dialContextFn dialContext, routeListener RouteListener) (DaxAPI, error) {
	endpoint := fmt.Sprintf("%s:%d", ip, port)
	return newSingleClientWithOptions(endpoint, connConfigData, region, credentials, maxPendingConnects, dialContextFn, routeListener)
}

type taskExecutor struct {
	tasks int32
	close chan struct{}
}

func newExecutor() *taskExecutor {
	return &taskExecutor{
		close: make(chan struct{}),
	}
}

func (e *taskExecutor) start(d time.Duration, action func() error) {
	ticker := time.NewTicker(d)
	atomic.AddInt32(&e.tasks, 1)
	go func() {
		for {
			select {
			case <-ticker.C:
				action() // TODO recover from panic()?
			case <-e.close:
				ticker.Stop()
				atomic.AddInt32(&e.tasks, -1)
				return
			}
		}
	}()
}

func (e *taskExecutor) numTasks() int32 {
	return atomic.LoadInt32(&e.tasks)
}

func (e *taskExecutor) stopAll() {
	close(e.close)
}

type RouteListener interface {
	isRouteManagerEnabled() bool
	addRoute(string, DaxAPI)
	removeRoute(string, DaxAPI)
}
