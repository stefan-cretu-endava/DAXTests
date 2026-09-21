package main

import (
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-dax-go-v2/dax/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func getDaxClient(cfg *aws.Config, f *flags, appConfig *AppConfig) (*dax.Dax, error) {
	if cfg == nil {
		panic("Unable to get aws.Config")
	}

	daxCfg := dax.NewConfig(*cfg, f.clusterEndpoint)
	// populate dax config
	daxCfg.Region = "us-east-1"
	daxCfg.SkipHostnameVerification = true
	daxCfg.MaxPendingConnectionsPerHost = int(appConfig.ClientConfig.MaxPendingConnections)
	daxCfg.ReadRetries = int(appConfig.ClientConfig.ReadRetries)
	daxCfg.WriteRetries = int(appConfig.ClientConfig.WriteRetries)
	daxCfg.DialContext = (&net.Dialer{
		Timeout:   time.Millisecond * time.Duration(appConfig.ClientConfig.ConnectionTimeout),
		KeepAlive: time.Minute,
	}).DialContext
	daxCfg.LogLevel = utils.LogOff // utils.LogDebug //utils.LogDebugWithRequestRetries

	daxCfg.ClientHealthCheckInterval = time.Second * 5
	daxCfg.ClusterUpdateInterval = time.Second * 5
	daxCfg.RouteManagerEnabled = true

	daxCfg.RequestTimeout = time.Duration(f.requestTimeoutMillis) * time.Millisecond
	//daxCfg.MaxDrainingTime = 100 * time.Millisecond

	fmt.Println("Created DAX client with request timeout:", daxCfg.RequestTimeout.String())

	client, err := dax.New(daxCfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}
