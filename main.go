package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type Data struct {
	Vpc                                       string   `json:"vpc,omitempty"`
	VpcCidar                                  string   `json:"vpc_cidar,omitempty"`
	VpcInstanceTenancy                        string   `json:"vpc_instance_tenancy"`
	InternetGateway                           string   `json:"internet_gateway,omitempty"`
	InternetGatewayAttachment                 string   `json:"internet_gateway_attachment,omitempty"`
	PublicRoute                               string   `json:"public_route,omitempty"`
	PublicRouteTable                          string   `json:"public_route_table,omitempty"`
	PrivateRouteTable                         string   `json:"private_route_table,omitempty"`
	PublicDestinationCidar                    string   `json:"public_destination_cidar,omitempty"`
	PublicSubnets                             []string `json:"public_subnets,omitempty"`
	PrivateSubnets                            []string `json:"private_subnets,omitempty"`
	AvailabilityZones                         []string `json:"availability_zones,omitempty"`
	PublicSubnetsPrefix                       string   `json:"public_subnets_prefix,omitempty"`
	PrivateSubnetPrefix                       string   `json:"private_subnets_prefix,omitempty"`
	PublicRouteTableSubnetsAssociationPrefix  string   `json:"public_route_table_subnets_association_prefix,omitempty"`
	PrivateRouteTableSubnetsAssociationPrefix string   `json:"private_route_table_subnets_association_prefix,omitempty"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		var configData Data
		cfg := config.New(ctx, "")
		cfg.RequireObject("config", &configData)

		// Create a VPC
		awsVpc, err := ec2.NewVpc(ctx, configData.Vpc, &ec2.VpcArgs{
			CidrBlock:       pulumi.String(configData.VpcCidar),
			InstanceTenancy: pulumi.String(configData.VpcInstanceTenancy),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(configData.Vpc),
			},
		})
		if err != nil {
			return err
		}

		// Create InternetGateway
		awsIGW, err := ec2.NewInternetGateway(ctx, configData.InternetGateway, &ec2.InternetGatewayArgs{
			Tags: pulumi.StringMap{
				"Name": pulumi.String(configData.InternetGateway),
			},
		})
		if err != nil {
			return err
		}

		// Attach InternetGateway to VPC
		_, err = ec2.NewInternetGatewayAttachment(ctx, configData.InternetGatewayAttachment, &ec2.InternetGatewayAttachmentArgs{
			InternetGatewayId: awsIGW.ID(),
			VpcId:             awsVpc.ID(),
		})
		if err != nil {
			return err
		}

		//cidrBlocksPublic := []string{"10.0.0.0/20", "10.0.16.0/20", "10.0.32.0/20"}
		//cidrBlocksPrivate := []string{"10.0.48.0/24", "10.0.64.0/24", "10.0.80.0/20"}
		//availabilityZones := []string{"us-east-1a", "us-east-1b", "us-east-1c"}

		var publicSubnets []pulumi.IDOutput
		var privateSubnets []pulumi.IDOutput

		// Iterate over cidrBlocksPublic to create a public subnets in each availability zones under an existing vpc
		for i, cidr := range configData.PublicSubnets {
			subnetName := fmt.Sprintf(configData.PublicSubnetsPrefix+"-%d", i)
			publicSubnet, err := ec2.NewSubnet(ctx, subnetName, &ec2.SubnetArgs{
				VpcId:            awsVpc.ID(),
				AvailabilityZone: pulumi.String(configData.AvailabilityZones[i]),
				CidrBlock:        pulumi.String(cidr),
				Tags: pulumi.StringMap{
					"Name": pulumi.String(subnetName),
				},
			})
			if err != nil {
				return err
			}

			publicSubnets = append(publicSubnets, publicSubnet.ID())
		}

		// Iterate over cidrBlocksPrivate to create a private subnets in each availability zones under an existing vpc
		for i, cidr := range configData.PrivateSubnets {
			subnetName := fmt.Sprintf(configData.PrivateSubnetPrefix+"-%d", i)
			privateSubnet, err := ec2.NewSubnet(ctx, subnetName, &ec2.SubnetArgs{
				VpcId:            awsVpc.ID(),
				AvailabilityZone: pulumi.String(configData.AvailabilityZones[i]),
				CidrBlock:        pulumi.String(cidr),
				Tags: pulumi.StringMap{
					"Name": pulumi.String(subnetName),
				},
			})
			if err != nil {
				return err
			}

			privateSubnets = append(privateSubnets, privateSubnet.ID())
		}

		// Create a public route table to an existing vpc
		//publicRouteTableName := "public-route-table"
		publicRouteTable, err := ec2.NewRouteTable(ctx, configData.PublicRouteTable, &ec2.RouteTableArgs{
			VpcId: awsVpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(configData.PublicRouteTable),
			},
		})
		if err != nil {
			return err
		}

		// Create a route for the public route table to an Internet Gateway (for public subnets)
		_, err = ec2.NewRoute(ctx, configData.PublicRoute, &ec2.RouteArgs{
			RouteTableId:         publicRouteTable.ID(),
			DestinationCidrBlock: pulumi.String(configData.PublicDestinationCidar),
			GatewayId:            awsIGW.ID(),
		})
		if err != nil {
			return err
		}

		// Associate the public route table with public subnets
		for i, publicSubnetID := range publicSubnets {
			_, err := ec2.NewRouteTableAssociation(ctx, fmt.Sprintf(configData.PublicRouteTableSubnetsAssociationPrefix+"-%d", i), &ec2.RouteTableAssociationArgs{
				SubnetId:     publicSubnetID,
				RouteTableId: publicRouteTable.ID(),
			})
			if err != nil {
				return err
			}
		}

		// Create separate route tables for private subnets
		var privateRouteTables []*ec2.RouteTable
		for i, privateSubnetID := range privateSubnets {
			routeTableName := fmt.Sprintf(configData.PrivateRouteTable+"-%d", i)
			privateRouteTable, err := ec2.NewRouteTable(ctx, routeTableName, &ec2.RouteTableArgs{
				VpcId: awsVpc.ID(),
				Tags: pulumi.StringMap{
					"Name": pulumi.String(routeTableName),
				},
			})
			if err != nil {
				return err
			}

			// Associate each private route table with its respective private subnet
			_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf(configData.PrivateRouteTableSubnetsAssociationPrefix+"-%d", i), &ec2.RouteTableAssociationArgs{
				SubnetId:     privateSubnetID,
				RouteTableId: privateRouteTable.ID(),
			})
			if err != nil {
				return err
			}

			privateRouteTables = append(privateRouteTables, privateRouteTable)
		}

		// Export the subnet and route table IDs for later use
		//ctx.Export("publicSubnetIDs", pulumi.ToStringArray(publicSubnets))
		//ctx.Export("privateSubnetIDs", pulumi.ToStringArray(privateSubnets))
		//ctx.Export("privateRouteTableIDs", pulumi.ToStringArray(pulumi.Map(privateRouteTables, func(rt *ec2.RouteTable) pulumi.IDOutput { return rt.ID() })))

		return nil
	})
}