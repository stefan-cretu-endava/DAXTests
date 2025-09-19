package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/smithy-go/logging"
)

type customLogger struct{}

func (l customLogger) Logf(classification logging.Classification, format string, v ...interface{}) {
	log.Printf("[%s] %s", classification, fmt.Sprintf(format, v...))
}

func getAwsConfig(region string) *aws.Config {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(getEnvVar("REGION", region)),
		config.WithLogger(customLogger{}),
		config.WithClientLogMode(aws.LogRetries|aws.LogRequest),
	)

	if err != nil {
		panic(err)
	}

	return &cfg

}

func getEnvVar(name, dflt string) string {
	out := os.Getenv(name)
	if out == "" {
		os.Getenv(fmt.Sprintf("AWS_%s", name))
	}
	if out == "" {
		return dflt
	}

	return out
}
