package main

import "fmt"

func main() {

	dualStackEndpointURL := "dax://csc-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	ipv6EndpointURL := "dax://csc-ipv6.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	ipv4EndpointURL := "dax://csc-cluster.cykcls.dax-clusters.eu-west-1.amazonaws.com"
	awsConfigUSEast1 := getAwsConfig("us-east-1")
	awsConfigEUWest1 := getAwsConfig("eu-west-1")

	// IPv6 ipDiscovery
	fmt.Println("--------------------Start testing scenarios for IPv6 ipDiscovery-----------------------")
	var ipDiscovery string = string("IPv6")
	fmt.Println("Start testing IPv4 cluster with ipDiscovery = " + ipDiscovery)
	ipv4DaxClientIPv6, err := getDaxClient(awsConfigEUWest1, ipv4EndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for IPv4 cluster with ipDiscovery = " + ipDiscovery)
	defer ipv4DaxClientIPv6.Close()

	fmt.Println("Start testing IPv6 cluster with ipDiscovery = " + ipDiscovery)
	ipv6DaxClientIPv6, err := getDaxClient(awsConfigUSEast1, ipv6EndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for IPv6 cluster with ipDiscovery = " + ipDiscovery)
	defer ipv6DaxClientIPv6.Close()

	fmt.Println("Start testing dual stack cluster with ipDiscovery = " + ipDiscovery)
	dualStackDaxClientIPv6, err := getDaxClient(awsConfigUSEast1, dualStackEndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for dual stack cluster with ipDiscovery = " + ipDiscovery)
	defer dualStackDaxClientIPv6.Close()

	// IPv4 ipDiscovery
	fmt.Println("--------------------Start testing scenarios for IPv4 ipDiscovery-----------------------")
	ipDiscovery = string("IPv4")
	fmt.Println("Start testing IPv4 cluster with ipDiscovery = " + ipDiscovery)
	ipv4DaxClientIPv4, err := getDaxClient(awsConfigEUWest1, ipv4EndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for IPv4 cluster with ipDiscovery = " + ipDiscovery)
	defer ipv4DaxClientIPv4.Close()

	fmt.Println("Start testing IPv6 cluster with ipDiscovery = " + ipDiscovery)
	ipv6DaxClientIPv4, err := getDaxClient(awsConfigUSEast1, ipv6EndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for IPv6 cluster with ipDiscovery = " + ipDiscovery)
	defer ipv6DaxClientIPv4.Close()

	fmt.Println("Start testing dual stack cluster with ipDiscovery = " + ipDiscovery)
	dualStackDaxClientIPv4, err := getDaxClient(awsConfigUSEast1, dualStackEndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for dual stack cluster with ipDiscovery = " + ipDiscovery)
	defer dualStackDaxClientIPv4.Close()

	// Empty ipDiscovery
	fmt.Println("--------------------Start testing scenarios for empty ipDiscovery-----------------------")
	ipDiscovery = string("")

	fmt.Println("Start testing IPv4 cluster with ipDiscovery = " + ipDiscovery)
	ipv4DaxClient, err := getDaxClient(awsConfigEUWest1, ipv4EndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for IPv4 cluster with ipDiscovery = " + ipDiscovery)
	defer ipv4DaxClient.Close()

	fmt.Println("Start testing IPv6 cluster with ipDiscovery = " + ipDiscovery)
	ipv6DaxClient, err := getDaxClient(awsConfigUSEast1, ipv6EndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for IPv6 cluster with ipDiscovery = " + ipDiscovery)
	defer ipv6DaxClient.Close()

	fmt.Println("Start testing dual stack cluster with ipDiscovery = " + ipDiscovery)
	dualStackDaxClient, err := getDaxClient(awsConfigUSEast1, dualStackEndpointURL, ipDiscovery)
	if err != nil {
		panic(err)
	}
	fmt.Println("Finished creating dax client for dual stack cluster with ipDiscovery = " + ipDiscovery)
	defer dualStackDaxClient.Close()

}
