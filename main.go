package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/rds"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := aws.GetRegion(ctx, &aws.GetRegionArgs{})
		if err != nil {
			return err
		}

		vpc, err := createVpcResources(ctx)
		if err != nil {
			return err
		}

		dbSubnetGroup, err := rds.NewSubnetGroup(ctx, "rds-subnet-group", &rds.SubnetGroupArgs{
			SubnetIds: vpc.PrivateSubnetIds,
			Tags: pulumi.StringMap{
				"Name": pulumi.String("rds-subnet-group"),
			},
		})
		if err != nil {
			return err
		}

		conf := config.New(ctx, "")
		dbPassword := conf.RequireSecret("dbPassword")

		dbInstance, err := rds.NewInstance(ctx, "rds-instance", &rds.InstanceArgs{
			Engine:              pulumi.String("mysql"),
			EngineVersion:       pulumi.String("8.4"),
			InstanceClass:       pulumi.String("db.t3.micro"),
			AllocatedStorage:    pulumi.Int(20),
			Username:            pulumi.String("admin"),
			Password:            dbPassword,
			MultiAz:             pulumi.Bool(true),
			DbSubnetGroupName:   dbSubnetGroup.Name,
			PubliclyAccessible:  pulumi.Bool(false),
			SkipFinalSnapshot:   pulumi.Bool(true),
			VpcSecurityGroupIds: pulumi.StringArray{vpc.DefaultSecurityGroupId},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("rds-multi-az"),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("vpcId", vpc.Vpc.ID())
		ctx.Export("rdsEndpoint", dbInstance.Endpoint)
		ctx.Export("rdsPort", dbInstance.Port)
		ctx.Export("rdsIdentifier", dbInstance.ID())

		return nil
	})
}
