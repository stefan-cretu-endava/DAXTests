package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func getAwsConfig() *aws.Config {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(getEnvVar("REGION", "us-east-1")),
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
