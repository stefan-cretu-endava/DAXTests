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

package dax

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"time"

	"github.com/aws/aws-dax-go-v2/dax/internal/client"
	"github.com/aws/aws-dax-go-v2/dax/internal/proxy"
	"github.com/aws/aws-dax-go-v2/dax/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go/logging"
)

// Dax makes requests to the Amazon DAX API, which conforms to the DynamoDB API.
//
// Dax methods are safe to use concurrently
type Dax struct {
	client client.DaxAPI
	config Config
}

const ServiceName = "dax"

type Config struct {
	client.Config

	// Default request options
	RequestTimeout time.Duration
	WriteRetries   int
	ReadRetries    int
	RetryMode      aws.RetryMode
	RetryDelay     time.Duration
	retryer        aws.Retryer

	Logger   logging.Logger
	LogLevel utils.LogLevelType
}

// DefaultConfig returns the default DAX configuration.
//
// Config.Region and Config.HostPorts still need to be configured properly
// to start up a DAX client.
func DefaultConfig() Config {
	return Config{
		Config:         client.DefaultConfig(),
		RequestTimeout: 1 * time.Minute,
		WriteRetries:   2,
		ReadRetries:    2,
		Logger:         utils.NewDefaultLogger(),
		LogLevel:       utils.LogOff,
		RetryDelay:     0 * time.Second,
	}
}

// NewConfig creates a new instance of the DAX config with an aws.Config.
func NewConfig(config aws.Config, endpoint string) Config {
	dc := DefaultConfig()
	dc.mergeFrom(config, endpoint)
	return dc
}

// New creates a new instance of the DAX client with a DAX configuration.
func New(cfg Config) (*Dax, error) {
	cfg.Config.SetLogger(cfg.Logger, cfg.LogLevel)
	c, err := client.New(cfg.Config)
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Logf("ERROR", "Exception in initialisation of DAX Client : %s", err)
		}
		return nil, err
	}
	return &Dax{client: c, config: cfg}, nil
}

// SecureDialContext creates a secure DialContext for connecting to encrypted cluster
func SecureDialContext(endpoint string, skipHostnameVerification bool) (func(ctx context.Context, network string, address string) (net.Conn, error), error) {
	dialer := &proxy.Dialer{}
	var cfg tls.Config
	if skipHostnameVerification {
		cfg = tls.Config{InsecureSkipVerify: true}
	} else {
		u, err := url.ParseRequestURI(endpoint)
		if err != nil {
			return nil, err
		}
		cfg = tls.Config{ServerName: u.Hostname()}
	}
	dialer.Config = &cfg
	return dialer.DialContext, nil
}

// NewFromConfig creates a new instance of the DAX client with an aws.Config.
//
// Example:
//
//	config := aws.Config{
//		Region: "us-east-1",
//	}
//
//	// Create a DAX client.
//	svc := dax.NewFromConfig(config, "dax://mycluster.frfx8h.clustercfg.dax.usw2.amazonaws.com:8111")
func NewFromConfig(config aws.Config, endpoint string) (*Dax, error) {
	dc := DefaultConfig()
	dc.mergeFrom(config, endpoint)
	return New(dc)
}

func (c *Config) mergeFrom(ac aws.Config, endpoint string) {
	if r := ac.RetryMaxAttempts; r > 0 {
		c.WriteRetries = r
		c.ReadRetries = r
	}
	if ac.Logger != nil {
		c.Logger = ac.Logger
	}

	if ac.Credentials != nil {
		c.Credentials = ac.Credentials
	}
	if endpoint != "" {
		c.HostPorts = []string{endpoint}
	}
	if ac.Region != "" {
		c.Region = ac.Region
	}
}

func (c *Config) requestOptions(read bool, ctx context.Context, optFns ...func(*dynamodb.Options)) (client.RequestOptions, context.CancelFunc, error) {
	r := c.WriteRetries
	if read {
		r = c.ReadRetries
	}
	var cfn context.CancelFunc
	if ctx == nil {
		ctx = context.Background()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline && c.RequestTimeout > 0 {
		ctx, cfn = context.WithTimeout(ctx, c.RequestTimeout)
	}
	opt := client.RequestOptions{}
	opt.Logger = c.Logger
	opt.LogLevel = c.LogLevel
	opt.RetryMaxAttempts = r
	opt.RetryDelay = c.RetryDelay
	opt.Context = ctx

	// merge from request options
	for _, o := range optFns {
		o(&opt.Options)
	}

	if err := client.RejectCustomMiddleware(opt.APIOptions); err != nil {
		return client.RequestOptions{}, cfn, err
	}

	return opt, cfn, nil
}
