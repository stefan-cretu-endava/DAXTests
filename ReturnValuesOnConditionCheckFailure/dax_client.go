package main

import (
	"net"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
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
	daxCfg.LogLevel = 2 // utils.LogDebugWithRequestRetries

	// populate client cluster config fields in dax config
	daxCfg.Config.SkipHostnameVerification = true
	daxCfg.Config.RouteManagerEnabled = true
	// daxCfg.Config.MaxPendingConnectionsPerHost = int(appConfig.ClientConfig.MaxPendingConnections)
	// daxCfg.Config.ClientHealthCheckInterval = time.Second * 5
	// daxCfg.Config.ClusterUpdateInterval = time.Second * 5
	daxCfg.Config.DialContext = (&net.Dialer{
		Timeout:   time.Millisecond * time.Duration(5000),
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
