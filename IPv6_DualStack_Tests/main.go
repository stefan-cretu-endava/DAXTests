package main

import (
	"fmt"

	"github.com/aws/aws-dax-go-v2/dax/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func createDaxClientWithIpDiscovery(awsConfig *aws.Config, endpointURL string, clusterType string, ipDiscovery utils.IpDiscovery) {
	fmt.Println("[Cluster type: " + clusterType + "] Creating dax client with ipDiscovery = " + ipDiscovery.String())
	daxClient, err := getDaxClient(awsConfig, endpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("[Cluster type: " + clusterType + "] Finished creating dax client with ipDiscovery = " + ipDiscovery.String())
	daxClient.Close()
}

func main() {

	dualStackEndpointURL := "dax://csc-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	ipv6EndpointURL := "dax://csc-ipv6.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	//ipv4EUEndpointURL := "dax://csc-cluster.cykcls.dax-clusters.eu-west-1.amazonaws.com"
	ipv4USndpointURL := "dax://csc-ipv4-ubuntu.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	awsConfigUSEast1 := getAwsConfig("us-east-1")
	//awsConfigEUWest1 := getAwsConfig("eu-west-1")

	// IPv4 cluster
	func() {
		fmt.Println("--------------------Start testing scenarios on IPv4 cluster-----------------------")
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv4USndpointURL, "IPv4", utils.IpDiscoveryIPv4)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv4USndpointURL, "IPv4", utils.IpDiscoveryIPv6)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv4USndpointURL, "IPv4", utils.IpDiscoveryEmpty)
		fmt.Println("--------------------Finished testing scenarios on IPv4 cluster-----------------------")
	}()

	// IPv6 cluster
	func() {
		fmt.Println("--------------------Start testing scenarios on IPv6 cluster-----------------------")
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv6EndpointURL, "IPv6", utils.IpDiscoveryIPv4)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv6EndpointURL, "IPv6", utils.IpDiscoveryIPv6)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, ipv6EndpointURL, "IPv6", utils.IpDiscoveryEmpty)
		fmt.Println("--------------------Finished testing scenarios on IPv6 cluster-----------------------")
	}()

	//Dual stack
	func() {
		fmt.Println("--------------------Start testing scenarios on dual stack config-----------------------")
		createDaxClientWithIpDiscovery(awsConfigUSEast1, dualStackEndpointURL, "Dual Stack (IPv4 + IPv6)", utils.IpDiscoveryIPv4)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, dualStackEndpointURL, "Dual Stack (IPv4 + IPv6)", utils.IpDiscoveryIPv6)
		createDaxClientWithIpDiscovery(awsConfigUSEast1, dualStackEndpointURL, "Dual Stack (IPv4 + IPv6)", utils.IpDiscoveryEmpty)
		fmt.Println("--------------------Finished testing scenarios on dual stack config-----------------------")
	}()
}
