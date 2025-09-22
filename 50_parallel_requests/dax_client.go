package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-dax-go-v2/dax/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func getDaxClient(cfg *aws.Config, endpoint string) (*dax.Dax, error) {
	if cfg == nil {
		panic("Unable to get aws.Config")
	}

	// NewConfig return service Config that encapsulates client (cluster) Config struct
	daxCfg := dax.NewConfig(*cfg, endpoint)
	// populate dax config fields
	// daxCfg.ReadRetries = int(appConfig.ClientConfig.ReadRetries)
	// daxCfg.WriteRetries = int(appConfig.ClientConfig.WriteRetries)
	// daxCfg.RequestTimeout = time.Millisecond * time.Duration(appConfig.ClientConfig.RequestTimeout)
	daxCfg.LogLevel = utils.LogDebugWithRequestRetries

	// populate client cluster config fields in dax config
	daxCfg.Config.SkipHostnameVerification = true
	daxCfg.Config.RouteManagerEnabled = true
	// daxCfg.Config.MaxPendingConnectionsPerHost = int(appConfig.ClientConfig.MaxPendingConnections)
	daxCfg.Config.ClientHealthCheckInterval = time.Second * 15
	daxCfg.Config.ClusterUpdateInterval = time.Second * 15
	daxCfg.Config.DialContext = (&net.Dialer{
		Timeout:   time.Millisecond * time.Duration(10000),
		KeepAlive: time.Minute,
	}).DialContext

	// The above created dax (service) Config is used to create a *Dax.
	// Internally it requires a client instance, for which new is called using daxCfg.Config (client cluster Config).
	// Here the control flow is service New->cluster New->cluster.start()->cluster.safeRefresh(false)->cluster.refresh(false)->cluster.refreshNow()
	client, err := dax.New(daxCfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getSecureDAXClientForTLS(cfg *aws.Config, endpoint string) (*dax.Dax, error) {
	if cfg == nil {
		panic("Unable to get aws.Config")
	}

	//Connecion to a secure cluster
	secureEndpoint := endpoint
	secureCfg := dax.NewConfig(*cfg, endpoint)
	secureCfg.HostPorts = []string{endpoint}
	secureCfg.Region = "us-east-1"
	secureCfg.LogLevel = utils.LogDebugWithRequestRetries
	secureCfg.Config.ClientHealthCheckInterval = time.Second * 15
	secureCfg.Config.ClusterUpdateInterval = time.Second * 15

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
