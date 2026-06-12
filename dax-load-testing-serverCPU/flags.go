package main

import (
	"flag"
	"strings"
)

const (
	ClusterEndpointIPv4 string = "dax://csc-ipv4.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	ClusterNameIPv4     string = "csc-ipv4"
	IPv4Namespace       string = "CSC-baseline-IPv4"

	TLSClusterEndpointIPv4 string = "daxs://csc-tls-ipv4.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	TLSClusterNameIPv4     string = "csc-tls-ipv4"

	ClusterEndpointIPv6 string = "dax://csc-ipv6.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	ClusterNameIPv6     string = "csc-ipv6"
	IPv6Namespace       string = "CSC-baseline-IPv6"

	ClusterEndpointDualStack string = "dax://csc-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	ClusterNameDualStack     string = "csc-dualstack"
	DualStackNamespace       string = "CSC-Dualstack"

	TLSClusterEndpointDualStack string = "daxs://csc-tls-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	TLSClusterNameDualStack     string = "csc-dualstack-tls"
)

type flags = struct {
	test                string
	op                  string
	clusterType         string
	isTLSEnabled        bool
	clusterName         string
	clusterEndpoint     string
	testNamespace       string
	testDurationMinutes int
}

func getFlags() *flags {
	f := &flags{}

	flag.StringVar(&f.test, "test", "", "")
	flag.StringVar(&f.op, "op", "read", "")
	flag.StringVar(&f.clusterType, "cluster-type", "ipv4", "")
	flag.BoolVar(&f.isTLSEnabled, "tls", false, "")
	flag.IntVar(&f.testDurationMinutes, "duration", 60, "")
	flag.Parse()

	if strings.EqualFold(f.clusterType, "ipv4") {
		if f.isTLSEnabled {
			f.clusterEndpoint = TLSClusterEndpointIPv4
			f.clusterName = TLSClusterNameIPv4
		} else {
			f.clusterEndpoint = ClusterEndpointIPv4
			f.clusterName = ClusterNameIPv4
		}
		f.testNamespace = /*"CSCDecoupleRT"*/ IPv4Namespace
	} else if strings.EqualFold(f.clusterType, "ipv6") {
		f.clusterEndpoint = ClusterEndpointIPv6
		f.clusterName = ClusterNameIPv6
		f.testNamespace = IPv6Namespace
	} else if strings.EqualFold(f.clusterType, "dualstack") || strings.EqualFold(f.clusterType, "dual-stack") || strings.EqualFold(f.clusterType, "dual_stack") {
		if f.isTLSEnabled {
			f.clusterEndpoint = TLSClusterEndpointDualStack
			f.clusterName = TLSClusterNameDualStack
		} else {
			f.clusterEndpoint = ClusterEndpointDualStack
			f.clusterName = ClusterNameDualStack
		}
		f.testNamespace = DualStackNamespace //"CSCDecoupleRT"
	}

	return f
}
