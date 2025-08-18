package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-dax-go-v2/dax/types"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func getDaxClient(cfg *aws.Config, aggressive bool) (*dax.Dax, error) {
	if cfg == nil {
		panic("Unable to get aws.Config")
	}

	daxCfg := dax.NewConfig(*cfg, Flags.AWS.DAX.Endpoint)
	// populate dax config
	daxCfg.SkipHostnameVerification = true
	daxCfg.RequestTimeout = time.Second
	daxCfg.MaxPendingConnectionsPerHost = int(Flags.AWS.DAX.MaxPendingConnectionAcquires)
	daxCfg.ReadRetries = ternary(aggressive, int(Flags.AWS.DAX.ReadRetriesAggressive), int(Flags.AWS.DAX.ReadRetries))
	daxCfg.WriteRetries = ternary(aggressive, int(Flags.AWS.DAX.WriteRetriesAggressive), int(Flags.AWS.DAX.WriteRetries))
	daxCfg.RequestTimeout = time.Millisecond * time.Duration(
		ternary(
			aggressive,
			Flags.AWS.DAX.RequestTimeoutAggressiveMillis,
			Flags.AWS.DAX.RequestTimeoutMillis,
		),
	)
	daxCfg.DialContext = (&net.Dialer{
		Timeout:   time.Millisecond * time.Duration(Flags.AWS.DAX.ConnectionTtlMillis),
		KeepAlive: time.Minute,
	}).DialContext
	daxCfg.LogLevel = 2 // utils.LogDebugWithRequestRetries
	//daxCfg.Config.IpDiscovery = types.IpDiscoveryIPv6

	// healthcheck yo!
	daxCfg.ClientHealthCheckInterval = time.Second * 5
	daxCfg.ClusterUpdateInterval = time.Second * 5
	daxCfg.RouteManagerEnabled = true

	client, err := dax.New(daxCfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getDefaultDaxClient(cfg *aws.Config) (*dax.Dax, error) {
	if cfg == nil {
		panic("Unable to get aws.Config")
	}

	daxCfg := dax.NewConfig(*cfg, Flags.AWS.DAX.Endpoint)
	// populate dax config
	////daxCfg.SkipHostnameVerification = true
	////daxCfg.MaxPendingConnectionsPerHost = int(appConfig.ClientConfig.MaxPendingConnections)
	////daxCfg.ReadRetries = int(appConfig.ClientConfig.ReadRetries)
	////daxCfg.WriteRetries = int(appConfig.ClientConfig.WriteRetries)
	//daxCfg.RequestTimeout = time.Millisecond * time.Duration(appConfig.ClientConfig.RequestTimeout)
	//daxCfg.DialContext = (&net.Dialer{
	//	Timeout:   time.Millisecond * time.Duration(appConfig.ClientConfig.ConnectionTimeout),
	//	KeepAlive: time.Minute,
	//}).DialContext
	daxCfg.LogLevel = 0 // utils.LogDebugWithRequestRetries
	//daxCfg.Config.IpDiscovery = types.IpDiscoveryIPv6

	// healthcheck yo!
	//daxCfg.ClientHealthCheckInterval = time.Second * 5
	//daxCfg.ClusterUpdateInterval = time.Second * 5
	//daxCfg.RouteManagerEnabled = true

	client, err := dax.New(daxCfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getSecureDAXClientForTLS(cfg *aws.Config, aggressive bool) (*dax.Dax, error) {
	if cfg == nil {
		panic("Unable to get aws.Config")
	}

	//Connecion to a secure cluster
	secureEndpoint := Flags.AWS.DAX.Endpoint
	secureCfg := dax.NewConfig(*cfg, Flags.AWS.DAX.Endpoint)
	secureCfg.HostPorts = []string{Flags.AWS.DAX.Endpoint}
	secureCfg.Region = "us-east-1"
	secureCfg.Config.IpDiscovery = types.IpDiscoveryIPv6

	secureCfg.SkipHostnameVerification = false

	secureCfg.DialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
		// fmt.Println("Write your custom logic here")
		dialCon, err := dax.SecureDialContext(secureEndpoint, secureCfg.SkipHostnameVerification)
		if err != nil {
			panic(fmt.Errorf("secure dialcontext creation failed %v", err))
		}
		return dialCon(ctx, network, address)
	}
	secureClient, err := dax.New(secureCfg)
	if err != nil {
		panic(fmt.Errorf("unable to initialize secure client %v", err))
	}
	fmt.Println("secure client created", secureClient)

	return secureClient, err
}
