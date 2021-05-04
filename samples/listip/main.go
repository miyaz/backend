package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	config := &aws.Config{
		Region: aws.String("ap-northeast-1"),
	}
	sess := session.Must(session.NewSession(config))
	svc := ec2metadata.New(sess)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if svc.AvailableWithContext(ctx) {
		doc, _ := svc.GetInstanceIdentityDocumentWithContext(ctx)
		fmt.Println(doc.Region)
		fmt.Println(doc.AvailabilityZone)
		fmt.Println(svc.GetMetadataWithContext(ctx, "/instance-type"))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/public-hostname"))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/public-ipv4"))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/local-hostname"))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/local-ipv4"))
		mac, _ := svc.GetMetadataWithContext(ctx, "/mac")
		//mac, _ := svc.GetMetadataWithContext(ctx, "/network/interfaces/macs")
		fmt.Println(svc.GetMetadataWithContext(ctx, "/network/interfaces/macs/"+mac))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/network/interfaces/macs/"+mac+"/vpc-id"))
	} else {
		fmt.Println("imds not avail")
	}

	ec2Svc := ec2.New(sess)
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String("vpc-ec55438e")},
			},
			&ec2.Filter{
				Name:   aws.String("status"),
				Values: []*string{aws.String("in-use")},
			},
		},
	}
	result, err := ec2Svc.DescribeNetworkInterfaces(params)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	for _, ni := range result.NetworkInterfaces {
		fmt.Println(*ni.PrivateIpAddress)
		if canConnect(*ni.PrivateIpAddress) {
			fmt.Println("can connect")
		} else {
			fmt.Println("can not connect")
		}
	}
}

func canConnect(ip string) bool {
	conn, err := net.DialTimeout("tcp", ip+":80", time.Second)
	if err != nil {
		// handle error
		return false
	}
	defer conn.Close()
	return true
}
