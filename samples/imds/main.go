package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

func main() {
	sess := session.Must(session.NewSession())
	svc := ec2metadata.New(sess)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if svc.AvailableWithContext(ctx) {
		doc, _ := svc.GetInstanceIdentityDocumentWithContext(ctx)
		fmt.Println(doc.AvailabilityZone)
		fmt.Println(svc.GetMetadataWithContext(ctx, "/instance-type"))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/public-hostname"))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/public-ipv4"))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/local-hostname"))
		fmt.Println(svc.GetMetadataWithContext(ctx, "/local-ipv4"))
	} else {
		fmt.Println("imds not avail")
	}
}
