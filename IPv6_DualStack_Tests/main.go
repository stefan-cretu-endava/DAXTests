package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-dax-go-v2/dax/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
)

func createDaxClientWithIpDiscovery(awsConfig *aws.Config, endpointURL string, clusterType string, ipDiscovery types.IpDiscovery) {
	fmt.Println("[Cluster type: " + clusterType + "] Creating dax client with ipDiscovery = " + ipDiscovery.String())
	daxClient, err := getDaxClient(awsConfig, endpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("[Cluster type: " + clusterType + "] Finished creating dax client with ipDiscovery = " + ipDiscovery.String())
	daxClient.Close()
}

func test() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	client := elasticache.NewFromConfig(cfg)

	input := &elasticache.CreateCacheClusterInput{
		CacheClusterId:           aws.String("csc-ipv6"),
		Engine:                   aws.String("redis"),
		EngineVersion:            aws.String("7.0"), // must be >= 6
		CacheNodeType:            aws.String("dax.r5.large"),
		NumCacheNodes:            aws.Int32(1),                // must be 1 for non-clustered mode
		CacheSubnetGroupName:     aws.String("ot-sng-ipv6-1"), // must support IPv6
		SecurityGroupIds:         []string{"sg-04edafd82030533ea"},
		TransitEncryptionEnabled: aws.Bool(true),
		IpDiscovery:              "some stuff",
	}

	output, err := client.CreateCacheCluster(context.TODO(), input)
	if err != nil {
		log.Fatalf("failed to create cache cluster: %v", err)
	}

	fmt.Printf("Created cache cluster: %s\n", *output.CacheCluster.CacheClusterId)
}

func main() {

	//test()

	dualStackEndpointURL := "dax://csc-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	ipv6EndpointURL := "dax://csc-ipv6.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	//ipv4EUEndpointURL := "dax://csc-cluster.cykcls.dax-clusters.eu-west-1.amazonaws.com"
	ipv4USndpointURL := "dax://csc-ipv4-ubuntu.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	awsConfigUSEast1 := getAwsConfig("us-east-1")
	//awsConfigEUWest1 := getAwsConfig("eu-west-1")

	// IPv4 cluster
	func() {
		fmt.Println("--------------------Start testing scenarios on IPv4 cluster-----------------------")
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv4USndpointURL, "IPv4", types.IpDiscoveryIPv4)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv4USndpointURL, "IPv4", types.IpDiscoveryIPv6)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv4USndpointURL, "IPv4", "")
		//createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv4USndpointURL, "IPv4", "random input supposed to be wrong")
		fmt.Println("--------------------Finished testing scenarios on IPv4 cluster-----------------------")
	}()

	// IPv6 cluster
	func() {
		fmt.Println("--------------------Start testing scenarios on IPv6 cluster-----------------------")
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv6EndpointURL, "IPv6", types.IpDiscoveryIPv4)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv6EndpointURL, "IPv6", types.IpDiscoveryIPv6)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv6EndpointURL, "IPv6", "")
		/*
		* [ERROR] Exception in initialisation of DAX Client : missing required field, config.IPDiscovery must be 'ipv4' or 'ipv6'.
		* panic: missing required field, config.IPDiscovery must be 'ipv4' or 'ipv6'.
		 */
		//createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv6EndpointURL, "IPv6", "ipv5")
		fmt.Println("--------------------Finished testing scenarios on IPv6 cluster-----------------------")
	}()

	//Dual stack
	func() {
		fmt.Println("--------------------Start testing scenarios on dual stack config-----------------------")
		createDaxClientWithIpDiscovery(awsConfigUSEast1, dualStackEndpointURL, "Dual Stack (IPv4 + IPv6)", types.IpDiscoveryIPv4)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, dualStackEndpointURL, "Dual Stack (IPv4 + IPv6)", types.IpDiscoveryIPv6)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, dualStackEndpointURL, "Dual Stack (IPv4 + IPv6)", "")
		//createDaxClientWithIpDiscovery(awsConfigUSEast1, dualStackEndpointURL, "Dual Stack (IPv4 + IPv6)", ".")
		fmt.Println("--------------------Finished testing scenarios on dual stack config-----------------------")
	}()
}
