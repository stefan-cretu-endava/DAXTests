package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func getAwsConfig(region string) *aws.Config {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(getEnvVar("REGION", region)),
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
