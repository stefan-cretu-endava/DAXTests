package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dax"
	"github.com/aws/aws-sdk-go-v2/service/dax/types"
)

const (
	subnet            = "radu-subnet"
	vpc               = "vpc-0bf793d12fb6c76e3"
	sg                = "sg-092785fb752ab49f0"
	role              = "Endava_DAX_Access_Role"
	roleArn           = "arn:aws:iam::886436938615:role/service-role/Endava_DAX_Access_Role"
	nodeType          = "dax.r5.large"
	replicationFactor = 3

	// parameter groups
	normalTtl = "performancetestconfiguration" // 30 days
	lowTtl    = "cachemiss"                    // 1 sec
	noTtl     = "ot-1ms-pg"                    // 1 ms
)

type DaxSvc struct {
	client *dax.Client
}

func (cm *DaxSvc) CreateCluster(name, parameterGroup string) string {
	_, err := cm.client.CreateCluster(context.Background(), &dax.CreateClusterInput{
		ClusterName:                   aws.String(name),
		IamRoleArn:                    aws.String(roleArn),
		NodeType:                      aws.String(nodeType),
		ReplicationFactor:             replicationFactor,
		ClusterEndpointEncryptionType: types.ClusterEndpointEncryptionTypeNone,
		Description:                   nil,
		ParameterGroupName:            aws.String(parameterGroup),
		SSESpecification: &types.SSESpecification{
			Enabled: aws.Bool(false),
		},
		SecurityGroupIds: []string{sg},
		SubnetGroupName:  aws.String(subnet),
	})

	if err != nil {
		panic(err)
	}

	maxNumberOfErrors := 15

	for {
		log.Println("waiting 15 seconds")
		<-time.After(time.Second * 15)

		s, err := cm.client.DescribeClusters(context.Background(), &dax.DescribeClustersInput{
			ClusterNames: []string{name},
		})
		if err != nil {
			maxNumberOfErrors--
			if maxNumberOfErrors < 1 {
				panic(err)
			}
		}

		if len(s.Clusters) != 1 {
			panic(fmt.Sprintf("wrong number of clusters: %d", len(s.Clusters)))
		}

		c := s.Clusters[0]

		if strings.EqualFold(aws.ToString(c.Status), "available") {
			log.Printf("Cluster ready: %s", name)
			log.Printf("Cluster endpoint: %s", *c.ClusterDiscoveryEndpoint.URL)

			return aws.ToString(c.ClusterDiscoveryEndpoint.URL)
		}
	}

	return ""
}

func (cm *DaxSvc) DeleteCluster(name string) {
	if name == "" {
		return
	}

	_, _ = cm.client.DeleteCluster(context.Background(), &dax.DeleteClusterInput{
		ClusterName: aws.String(name),
	})
}

func (cm *DaxSvc) RebootRandomNode(name string) {
	res, err := cm.client.DescribeClusters(context.Background(), &dax.DescribeClustersInput{
		ClusterNames: []string{name},
	})
	if err != nil {
		panic(err)
	}

	for _, cluster := range res.Clusters {
		nodeIds := []string{}
		for _, n := range cluster.Nodes {
			nodeIds = append(nodeIds, aws.ToString(n.NodeId))
		}

		rand.Seed(time.Now().UnixNano())
		idx := rand.Intn(len(nodeIds))

		_, _ = cm.client.RebootNode(context.Background(), &dax.RebootNodeInput{
			ClusterName: cluster.ClusterName,
			NodeId:      aws.String(nodeIds[idx]),
		})
	}
}

func getDaxSvc(cfg *aws.Config) *DaxSvc {
	return &DaxSvc{
		client: dax.NewFromConfig(*cfg),
	}
}
