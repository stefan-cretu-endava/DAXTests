package main

import (
	"flag"
	"fmt"
	"strings"
)

const (
	ClusterEndpointIPv4 string = "dax://csc-ipv4.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com" //"dax://liv-dax-large-cluster.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"
	ClusterNameIPv4     string = "csc-ipv4"                                                         //"liv-dax-large-cluster"
	IPv4Namespace       string = "CSC-baseline-IPv4"

	ClusterEndpointIPv45nodes string = "dax://csc-ipv4-medium-sized-a.6lzwui.nodes.alpha-dax-clusters.us-east-1.amazonaws.com"
	ClusterNameIPv45nodes     string = "csc-ipv4-medium-sized"

	ClusterEndpointIPv48nodes string = "dax://csc-ipv4-8nodes.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com" //"dax://csc-ipv4-large.lrjkec.gamma-dax-clusters.us-east-1.amazonaws.com"
	ClusterNameIPv48nodes     string = "csc-ipv4-8nodes"                                                         //"csc-ipv4-large"

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

type flags struct {
	test                 string
	op                   string
	clusterType          string
	isTLSEnabled         bool
	clusterName          string
	clusterEndpoint      string
	testNamespace        string
	testDurationMinutes  int
	requestTimeoutMillis int
	nodes                int
}

func getFlags() *flags {
	f := &flags{}

	flag.StringVar(&f.test, "test", "", "")
	flag.StringVar(&f.op, "op", "read", "")
	flag.StringVar(&f.clusterType, "clusterType", "ipv4", "")
	flag.IntVar(&f.nodes, "nodes", 3, "")
	flag.BoolVar(&f.isTLSEnabled, "tls", false, "")
	flag.IntVar(&f.testDurationMinutes, "duration", 60, "")
	flag.IntVar(&f.requestTimeoutMillis, "requestTimeoutMillis", 60000, "")
	flag.Parse()

	fmt.Println("getFlags set request timeout", f.requestTimeoutMillis)

	if strings.EqualFold(f.clusterType, "ipv4") {
		if f.isTLSEnabled {
			f.clusterEndpoint = TLSClusterEndpointIPv4
			f.clusterName = TLSClusterNameIPv4
		} else {
			switch f.nodes {
			case 5:
				f.clusterEndpoint = ClusterEndpointIPv45nodes
				f.clusterName = ClusterNameIPv45nodes
			case 8:
				f.clusterEndpoint = ClusterEndpointIPv48nodes
				f.clusterName = ClusterNameIPv48nodes
			default:
				f.clusterEndpoint = ClusterEndpointIPv4
				f.clusterName = ClusterNameIPv4
			}
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
