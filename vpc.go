package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VpcResources struct {
	Vpc                    *ec2.Vpc
	PublicSubnetId         pulumi.StringOutput
	PrivateSubnetIds       pulumi.StringArray
	DefaultSecurityGroupId pulumi.StringOutput
}

func getCurrentIP() (string, error) {
	resp, err := http.Get("https://checkip.amazonaws.com")
	if err != nil {
		return "", fmt.Errorf("error getting public IP: %v", err)
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading IP response: %v", err)
	}

	return fmt.Sprintf("%s/32", strings.TrimSpace(string(ip))), nil
}

func createVpcResources(ctx *pulumi.Context) (*VpcResources, error) {

	currentIP, err := getCurrentIP()
	if err != nil {
		return nil, fmt.Errorf("error getting current IP: %v", err)
	}

	vpc, err := ec2.NewVpc(ctx, "rds-vpc", &ec2.VpcArgs{
		CidrBlock:          pulumi.String("10.0.0.0/16"),
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		Tags: pulumi.StringMap{
			"Name": pulumi.String("rds-vpc"),
		},
	})
	if err != nil {
		return nil, err
	}

	igw, err := ec2.NewInternetGateway(ctx, "rds-igw", &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
		Tags: pulumi.StringMap{
			"Name": pulumi.String("rds-igw"),
		},
	})
	if err != nil {
		return nil, err
	}

	publicSubnet, err := ec2.NewSubnet(ctx, "public-subnet", &ec2.SubnetArgs{
		VpcId:            vpc.ID(),
		CidrBlock:        pulumi.String("10.0.1.0/24"),
		AvailabilityZone: pulumi.String("us-east-1a"),
		Tags: pulumi.StringMap{
			"Name": pulumi.String("rds-public-subnet"),
		},
	})
	if err != nil {
		return nil, err
	}

	privateSubnet1, err := ec2.NewSubnet(ctx, "private-subnet-1", &ec2.SubnetArgs{
		VpcId:            vpc.ID(),
		CidrBlock:        pulumi.String("10.0.2.0/24"),
		AvailabilityZone: pulumi.String("us-east-1a"),
		Tags: pulumi.StringMap{
			"Name": pulumi.String("rds-private-subnet-1"),
		},
	})
	if err != nil {
		return nil, err
	}

	privateSubnet2, err := ec2.NewSubnet(ctx, "private-subnet-2", &ec2.SubnetArgs{
		VpcId:            vpc.ID(),
		CidrBlock:        pulumi.String("10.0.3.0/24"),
		AvailabilityZone: pulumi.String("us-east-1b"),
		Tags: pulumi.StringMap{
			"Name": pulumi.String("rds-private-subnet-2"),
		},
	})
	if err != nil {
		return nil, err
	}

	publicRT, err := ec2.NewRouteTable(ctx, "public-rt", &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: igw.ID(),
			},
		},
		Tags: pulumi.StringMap{
			"Name": pulumi.String("rds-public-rt"),
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = ec2.NewRouteTableAssociation(ctx, "public-rta", &ec2.RouteTableAssociationArgs{
		SubnetId:     publicSubnet.ID(),
		RouteTableId: publicRT.ID(),
	})
	if err != nil {
		return nil, err
	}

	securityGroup, err := ec2.NewSecurityGroup(ctx, "rds-sg", &ec2.SecurityGroupArgs{
		VpcId: vpc.ID(),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(3306),
				ToPort:     pulumi.Int(3306),
				CidrBlocks: pulumi.StringArray{pulumi.String(currentIP)},
			},
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(3306),
				ToPort:     pulumi.Int(3306),
				CidrBlocks: pulumi.StringArray{pulumi.String(currentIP)},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: pulumi.StringMap{
			"Name": pulumi.String("rds-sg"),
		},
	})
	if err != nil {
		return nil, err
	}

	return &VpcResources{
		Vpc:                    vpc,
		PublicSubnetId:         publicSubnet.ID().ToStringOutput(),
		PrivateSubnetIds:       pulumi.StringArray{privateSubnet1.ID(), privateSubnet2.ID()},
		DefaultSecurityGroupId: securityGroup.ID().ToStringOutput(),
	}, nil
}
