package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/sns"
	"os"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/autoscaling"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/rds"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type EC2Instance struct {
	InstanceName             string `json:"instance_name,omitempty"`
	InstanceType             string `json:"instance_type,omitempty"`
	VolumeSize               int    `json:"volume_size,omitempty"`
	VolumeType               string `json:"volume_type,omitempty"`
	DeleteOnTermination      bool   `json:"delete_on_termination,omitempty"`
	DisableApiTermination    bool   `json:"disable_api_termination,omitempty"`
	AssociatePublicIpAddress bool   `json:"associate_public_ip,omitempty"`
	DeviceType               string `json:"device_type,omitempty"`
	AmiID                    string `json:"ami_id,omitempty"`
	SSHKeyName               string `json:"ssh_key_name,omitempty"`
	LogFilePath              string `json:"log_file_path,omitempty"`
	MetricServerPort         int    `json:"metric_server_port,omitempty"`
	UserDataFilePath         string `json:"users_data_file_path,omitempty"`
	MigrationsFilePath       string `json:"migrations_file_path,omitempty"`
	PublicKeyFilePath        string `json:"public_key_file_path,omitempty"`
}

type Resource struct {
	Region         string `json:"region"`
	AccountID      string `json:"account_id"`
	SNSTopic       string `json:"sns_topic"`
	SQSName        string `json:"sqs_name"`
	CertificateArn string `json:"certificate_arn"`
}

type MailerClient struct {
	APIKey string `json:"api_key,omitempty"`
	Name   string `json:"name,omitempty"`
	Email  string `json:"email,omitempty"`
	Domain string `json:"domain,omitempty"`
}

type DNS struct {
	ARecordName  string `json:"a_record_name,omitempty"`
	Type         string `json:"type,omitempty"`
	Ttl          int    `json:"ttl,omitempty"`
	Domain       string `json:"domain,omitempty"`
	HostedZoneID string `json:"hosted_zone_id,omitempty"`
}

type RDSInstance struct {
	SubnetGrp          string `json:"private_subnet_group,omitempty"`
	SecurityGroupName  string `json:"security_group_name,omitempty"`
	AllowsPort         int    `json:"allows_port,omitempty"`
	Protocol           string `json:"protocol,omitempty"`
	InstanceName       string `json:"instance_name,omitempty"`
	Engine             string `json:"engine,omitempty"`
	EngineVersion      string `json:"engine_version,omitempty"`
	InstanceClass      string `json:"instance_class,omitempty"`
	AllowedStorage     int    `json:"allowed_storage,omitempty"`
	Identifier         string `json:"identifier,omitempty"`
	Username           string `json:"username,omitempty"`
	Password           string `json:"password,omitempty"`
	DbName             string `json:"db_name,omitempty"`
	DbDriver           string `json:"db_driver,omitempty"`
	PubliclyAccessible bool   `json:"publicly_accessible,omitempty"`
	MultiAz            bool   `json:"multi_az,omitempty"`
	SkipFinalSnapShot  bool   `json:"skip_final_snapshot,omitempty"`
	StorageEncrypted   bool   `json:"storage_encrypted,omitempty"`
}

type Data struct {
	Vpc                                       string            `json:"vpc,omitempty"`
	VpcCidar                                  string            `json:"vpc_cidar,omitempty"`
	VpcInstanceTenancy                        string            `json:"vpc_instance_tenancy"`
	InternetGateway                           string            `json:"internet_gateway,omitempty"`
	InternetGatewayAttachment                 string            `json:"internet_gateway_attachment,omitempty"`
	PublicRoute                               string            `json:"public_route,omitempty"`
	PublicRouteTable                          string            `json:"public_route_table,omitempty"`
	PrivateRouteTable                         string            `json:"private_route_table,omitempty"`
	PublicDestinationCidar                    string            `json:"public_destination_cidar,omitempty"`
	PublicSubnets                             []string          `json:"public_subnets,omitempty"`
	PrivateSubnets                            []string          `json:"private_subnets,omitempty"`
	BitsToMask                                int               `json:"bits_to_mask,omitempty"`
	MaxAvailabilityZones                      int               `json:"max_availability_zones,omitempty"`
	AvailabilityZones                         []string          `json:"availability_zones,omitempty"`
	PublicSubnetsPrefix                       string            `json:"public_subnets_prefix,omitempty"`
	PrivateSubnetPrefix                       string            `json:"private_subnets_prefix,omitempty"`
	SecurityGroup                             string            `json:"security_group,omitempty"`
	SecurityRuleType                          string            `json:"security_rule_type,omitempty"`
	SecurityRuleProtocol                      string            `json:"security_rule_protocol,omitempty"`
	SecurityRuleNames                         map[string]string `json:"security_rule_names,omitempty"`
	InboundPorts                              map[string]int    `json:"all_inbound_ports,omitempty"`
	FetchPublicIPURL                          string            `json:"url_to_fetch_public_ip,omitempty"`
	EC2InstanceMetadata                       EC2Instance       `json:"ec2_instance_metadata,omitempty"`
	Dns                                       DNS               `json:"dns,omitempty"`
	ResourceParams                            Resource          `json:"resource_params"`
	RDSInstanceMetadata                       RDSInstance       `json:"rds_instance_metadata,omitempty"`
	MailerClientCreds                         MailerClient      `json:"mailer_client_crds,omitempty"`
	PublicRouteTableSubnetsAssociationPrefix  string            `json:"public_route_table_subnets_association_prefix,omitempty"`
	PrivateRouteTableSubnetsAssociationPrefix string            `json:"private_route_table_subnets_association_prefix,omitempty"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Load configuration values from pulumi.*.yaml file
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

		// Check for availability zones based on the region
		availableZones, err := aws.GetAvailabilityZones(ctx, nil, nil)
		if err != nil {
			return err
		}

		// Validation on MaxAvailabilityZones
		if configData.MaxAvailabilityZones == 0 || configData.MaxAvailabilityZones > len(availableZones.Names) {
			return errors.New(`{"status": 400, "msg": "Not sufficient AvailabilityZones"}`)
		}

		// Validation on BitsToMask
		if configData.BitsToMask == 0 || configData.BitsToMask > 32 {
			return errors.New(`{"status": 400, "msg": "Incorrect param bits_to_mask."}`)
		}

		// Assign AvailabilityZones based on our config requirements
		if availableZones != nil {
			if len(availableZones.Names) > configData.MaxAvailabilityZones {
				configData.AvailabilityZones = availableZones.Names[0:configData.MaxAvailabilityZones]
			} else {
				configData.AvailabilityZones = availableZones.Names
			}
		}

		// Calculate subnet cidr given VPC cidr, number of subnets required and bits to mask.
		subnets, err := CalculateCIDRSubnets(configData.VpcCidar, len(configData.AvailabilityZones)*2, configData.BitsToMask)
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

		var publicSubnets []pulumi.IDOutput
		var privateSubnets []pulumi.IDOutput

		// Iterate over cidrBlocksPublic to create a public subnets in each availability zones under an existing vpc
		noOfAvailabilityZones := len(configData.AvailabilityZones)
		for i := 0; i < noOfAvailabilityZones; i++ {
			cidr := subnets[i]
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
		for i := 0; i < noOfAvailabilityZones; i++ {
			cidr := subnets[noOfAvailabilityZones+i]
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
		publicRouteTable, err := ec2.NewRouteTable(ctx, configData.PublicRouteTable, &ec2.RouteTableArgs{
			VpcId: awsVpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(configData.PublicRouteTable),
			},
		})
		if err != nil {
			return err
		}

		// Create a private route table to an existing vpc
		privateRouteTable, err := ec2.NewRouteTable(ctx, configData.PrivateRouteTable, &ec2.RouteTableArgs{
			VpcId: awsVpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(configData.PrivateRouteTable),
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
		for i, privateSubnetID := range privateSubnets {
			// Associate each private route table with its respective private subnet
			_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf(configData.PrivateRouteTableSubnetsAssociationPrefix+"-%d", i), &ec2.RouteTableAssociationArgs{
				SubnetId:     privateSubnetID,
				RouteTableId: privateRouteTable.ID(),
			})
			if err != nil {
				return err
			}
		}

		// Fetch the public IP address of the system and allow only that IP to connect through SSH
		systemPublicIP, err := getPublicIPV4(configData.FetchPublicIPURL)
		if err != nil {
			return err
		}

		systemPublicIP = systemPublicIP + "/32"

		// Create a new security group for load balancer
		loadBalancerSecurityGroup, err := ec2.NewSecurityGroup(ctx, "load-balancer-security-group", &ec2.SecurityGroupArgs{
			VpcId: awsVpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("load-balancer-security-group"),
			},
			Ingress: ec2.SecurityGroupIngressArray{
				//&ec2.SecurityGroupIngressArgs{
				//	Description: pulumi.String("Allow inbound HTTP traffic on port 80 from all public IP addresses"),
				//	FromPort:    pulumi.Int(configData.InboundPorts["http"]),
				//	ToPort:      pulumi.Int(configData.InboundPorts["http"]),
				//	Protocol:    pulumi.String(configData.SecurityRuleProtocol),
				//	CidrBlocks:  pulumi.StringArray{pulumi.String(configData.PublicDestinationCidar)},
				//},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("Allow inbound HTTPS traffic on port 443 from all public IP addresses"),
					FromPort:    pulumi.Int(configData.InboundPorts["https"]),
					ToPort:      pulumi.Int(configData.InboundPorts["https"]),
					Protocol:    pulumi.String(configData.SecurityRuleProtocol),
					CidrBlocks:  pulumi.StringArray{pulumi.String(configData.PublicDestinationCidar)},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort:   pulumi.Int(8080),
					ToPort:     pulumi.Int(8080),
					Protocol:   pulumi.String(configData.SecurityRuleProtocol),
					CidrBlocks: pulumi.StringArray{pulumi.String(configData.PublicDestinationCidar)},
				},
				//&ec2.SecurityGroupEgressArgs{
				//	FromPort:   pulumi.Int(0),
				//	ToPort:     pulumi.Int(0),
				//	Protocol:   pulumi.String("-1"),
				//	CidrBlocks: pulumi.StringArray{pulumi.String(configData.PublicDestinationCidar)},
				//},
			},
		})
		if err != nil {
			return err
		}

		// Create a new security group for application running in EC2
		appSecurityGroup, err := ec2.NewSecurityGroup(ctx, configData.SecurityGroup, &ec2.SecurityGroupArgs{
			VpcId: awsVpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(configData.SecurityGroup),
			},
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("Allow inbound HTTPS traffic on port 8080 from public all public IP addresses"),
					FromPort:    pulumi.Int(configData.InboundPorts["customPort"]),
					ToPort:      pulumi.Int(configData.InboundPorts["customPort"]),
					Protocol:    pulumi.String(configData.SecurityRuleProtocol),
					SecurityGroups: pulumi.StringArray{
						loadBalancerSecurityGroup.ID(),
					},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("Allow inbound SSH traffic on port 22 from custom IP"),
					FromPort:    pulumi.Int(configData.InboundPorts["ssh"]),
					ToPort:      pulumi.Int(configData.InboundPorts["ssh"]),
					Protocol:    pulumi.String(configData.SecurityRuleProtocol),
					SecurityGroups: pulumi.StringArray{
						loadBalancerSecurityGroup.ID(),
					},
					//CidrBlocks: pulumi.StringArray{pulumi.String(configData.PublicDestinationCidar)},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					Protocol:   pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{pulumi.String(configData.PublicDestinationCidar)},
				},
			},
		})
		if err != nil {
			return err
		}

		// Create a custom parameter group to configure custom RDS Instance
		rdsParameterGroup, err := rds.NewParameterGroup(ctx, "webapp-parameter-group", &rds.ParameterGroupArgs{
			Description: pulumi.String("Custom parameter group for webapp rds instance"),
			Family:      pulumi.String("postgres15"),
			Name:        pulumi.String("webapp-rds-parameter-group"),
			Parameters: rds.ParameterGroupParameterArray{
				&rds.ParameterGroupParameterArgs{
					Name:  pulumi.String("rds.force_ssl"),
					Value: pulumi.String("0"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create a custom security group for the RDS instance.
		databaseSecurityGroup, err := ec2.NewSecurityGroup(ctx, configData.RDSInstanceMetadata.SecurityGroupName, &ec2.SecurityGroupArgs{
			VpcId:       awsVpc.ID(),
			Description: pulumi.String("Custom RDS Security Group"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("database-security-group"),
			},
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("Allow traffic from resources that use appSecurityGroup through 5432 port"),
					FromPort:    pulumi.Int(configData.RDSInstanceMetadata.AllowsPort),
					ToPort:      pulumi.Int(configData.RDSInstanceMetadata.AllowsPort),
					Protocol:    pulumi.String(configData.RDSInstanceMetadata.Protocol),
					SecurityGroups: pulumi.StringArray{
						appSecurityGroup.ID(),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewSecurityGroupRule(ctx, "AllowOutboundToDB", &ec2.SecurityGroupRuleArgs{
			Type:                  pulumi.String("egress"),
			FromPort:              pulumi.Int(configData.RDSInstanceMetadata.AllowsPort),
			ToPort:                pulumi.Int(configData.RDSInstanceMetadata.AllowsPort),
			Protocol:              pulumi.String(configData.RDSInstanceMetadata.Protocol),
			SourceSecurityGroupId: databaseSecurityGroup.ID(),
			SecurityGroupId:       appSecurityGroup.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewSecurityGroupRule(ctx, "AllowOutboundToInternet", &ec2.SecurityGroupRuleArgs{
			Type:            pulumi.String("egress"),
			FromPort:        pulumi.Int(443),
			ToPort:          pulumi.Int(443),
			Protocol:        pulumi.String(configData.RDSInstanceMetadata.Protocol),
			CidrBlocks:      pulumi.StringArray{pulumi.String(configData.PublicDestinationCidar)},
			SecurityGroupId: appSecurityGroup.ID(),
		})
		if err != nil {
			return err
		}

		// create a Subnet Group for all private subnets under a VPC.
		privateSubnetsStrs := make(pulumi.StringArray, len(privateSubnets))
		publicSubnetsStrs := make(pulumi.StringArray, len(publicSubnets))
		for _, subnet := range privateSubnets {
			privateSubnetsStrs = append(privateSubnetsStrs, subnet.ToIDOutput())
		}

		for _, subnet := range publicSubnets {
			publicSubnetsStrs = append(publicSubnetsStrs, subnet.ToIDOutput())
		}

		privateRDSSubnetGrp, err := rds.NewSubnetGroup(ctx, configData.RDSInstanceMetadata.SubnetGrp, &rds.SubnetGroupArgs{
			SubnetIds: privateSubnetsStrs,
			Tags: pulumi.StringMap{
				"Name": pulumi.String("database-private-subnet-grp"),
			},
		})
		if err != nil {
			return err
		}

		// Create the RDS instance with the custom security group and parameter group.
		rdsInstance, err := rds.NewInstance(ctx, configData.RDSInstanceMetadata.InstanceName, &rds.InstanceArgs{
			Engine:             pulumi.String(configData.RDSInstanceMetadata.Engine),
			EngineVersion:      pulumi.String(configData.RDSInstanceMetadata.EngineVersion),
			InstanceClass:      pulumi.String(configData.RDSInstanceMetadata.InstanceClass),
			AllocatedStorage:   pulumi.Int(configData.RDSInstanceMetadata.AllowedStorage),
			ApplyImmediately:   pulumi.Bool(true),
			Identifier:         pulumi.String(configData.RDSInstanceMetadata.Identifier),
			Username:           pulumi.String(configData.RDSInstanceMetadata.Username),
			Password:           pulumi.String(configData.RDSInstanceMetadata.Password),
			DbName:             pulumi.String(configData.RDSInstanceMetadata.DbName),
			ParameterGroupName: pulumi.StringPtrInput(rdsParameterGroup.Name),
			DbSubnetGroupName:  pulumi.StringInput(privateRDSSubnetGrp.Name),
			PubliclyAccessible: pulumi.Bool(configData.RDSInstanceMetadata.PubliclyAccessible),
			MultiAz:            pulumi.Bool(configData.RDSInstanceMetadata.MultiAz),
			SkipFinalSnapshot:  pulumi.Bool(configData.RDSInstanceMetadata.SkipFinalSnapShot),
			StorageEncrypted:   pulumi.Bool(configData.RDSInstanceMetadata.StorageEncrypted),
			VpcSecurityGroupIds: pulumi.StringArray{
				databaseSecurityGroup.ID(),
			},
		})
		if err != nil {
			return err
		}

		rdsInstance.Endpoint.ApplyT(func(rdsEndpoint string) error {
			rdsEndpoint = strings.Split(rdsEndpoint, ":")[0]

			// Read the public key content from the file.
			publicKeyContent, err := os.ReadFile(configData.EC2InstanceMetadata.PublicKeyFilePath)
			if err != nil {
				return err
			}

			// Create an EC2 key pair.
			_, err = ec2.NewKeyPair(ctx, configData.EC2InstanceMetadata.SSHKeyName, &ec2.KeyPairArgs{
				KeyName:   pulumi.String(configData.EC2InstanceMetadata.SSHKeyName),
				PublicKey: pulumi.String(publicKeyContent),
			})
			if err != nil {
				return err
			}

			// Create IAM role for EC2 instance
			ec2CloudWatchRoleStr, err := json.Marshal(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Effect": "Allow",
						"Action": []string{"sts:AssumeRole"},
						"Principal": map[string]interface{}{
							"Service": []string{"ec2.amazonaws.com"},
						},
					},
				},
			})

			role, err := iam.NewRole(ctx, "ec2CloudWatchRole", &iam.RoleArgs{
				AssumeRolePolicy: pulumi.String(ec2CloudWatchRoleStr),
				Tags: pulumi.StringMap{
					"tag-key": pulumi.String("ec2-cloudwatch-role"),
				},
			})
			if err != nil {
				return err
			}

			// Attach CloudWatchAgentServerPolicy to the new role
			_, err = iam.NewRolePolicyAttachment(ctx, "ec2CloudWatchPolicy", &iam.RolePolicyAttachmentArgs{
				Role:      role.ID(),
				PolicyArn: pulumi.String("arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"),
			})
			if err != nil {
				return err
			}

			snsAccessPolicy, err := json.Marshal(map[string]interface{}{
				"Version": "2008-10-17",
				"Id":      "__default_policy_ID",
				"Statement": []map[string]interface{}{
					{
						"Sid":    "__default_statement_ID",
						"Effect": "Allow",
						"Principal": map[string]string{
							"AWS": "*",
						},
						"Action": []string{
							"SNS:Publish",
							"SNS:RemovePermission",
							"SNS:SetTopicAttributes",
							"SNS:DeleteTopic",
							"SNS:ListSubscriptionsByTopic",
							"SNS:GetTopicAttributes",
							"SNS:AddPermission",
							"SNS:Subscribe",
						},
						"Resource": fmt.Sprintf("arn:aws:sns:%v:%v:%v", configData.ResourceParams.Region, configData.ResourceParams.AccountID, configData.ResourceParams.SNSTopic),
						"Condition": map[string]interface{}{
							"StringEquals": map[string]string{
								"AWS:SourceOwner": configData.ResourceParams.AccountID,
							},
						},
					},
					//{
					//	"Sid":    "__console_pub_0",
					//	"Effect": "Allow",
					//	"Principal": map[string]interface{}{
					//		"AWS": configData.ResourceParams.AccountID,
					//	},
					//	"Action":   "SNS:Publish",
					//	"Resource": fmt.Sprintf("arn:aws:sns:%v:%v:%v", configData.ResourceParams.Region, configData.ResourceParams.AccountID, configData.ResourceParams.SNSTopic),
					//},
					//{
					//	"Sid":    "__console_sub_0",
					//	"Effect": "Allow",
					//	"Principal": map[string]interface{}{
					//		"AWS": configData.ResourceParams.AccountID,
					//	},
					//	"Action": []string{
					//		"SNS:Subscribe",
					//	},
					//	"Resource": fmt.Sprintf("arn:aws:sns:%v:%v:%v", configData.ResourceParams.Region, configData.ResourceParams.AccountID, configData.ResourceParams.SNSTopic),
					//},
				},
			})
			if err != nil {
				return err
			}

			_, err = sns.NewTopic(ctx, "testSNSTopic", &sns.TopicArgs{
				Name:        pulumi.String(configData.ResourceParams.SNSTopic),
				DisplayName: pulumi.String("submissions"),
				FifoTopic:   pulumi.Bool(false),
				Policy:      pulumi.String(snsAccessPolicy),
			})
			if err != nil {
				return err
			}

			snsPolicyStr, err := json.Marshal(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Sid":    "VisualEditor0",
						"Effect": "Allow",
						"Action": []string{
							"sns:Publish",
							"sns:Subscribe",
						},
						"Resource": fmt.Sprintf("arn:aws:sns:%v:%v:%v", configData.ResourceParams.Region, configData.ResourceParams.AccountID, configData.ResourceParams.SNSTopic),
					},
					{
						"Sid": "VisualEditor1",
						"Action": []string{
							"sns:ListTopics",
							"sns:Unsubscribe",
							"sns:ListSubscriptions",
						},
						"Effect":   "Allow",
						"Resource": "*",
					},
				},
			})
			if err != nil {
				return err
			}

			customSNSPolicy, err := iam.NewPolicy(ctx, "CustomSNSPolicy2", &iam.PolicyArgs{
				Path:        pulumi.String("/"),
				Description: pulumi.String("Custom SNS policy to publish message from EC2 to SNS Topic"),
				Policy:      pulumi.String(snsPolicyStr),
			})
			if err != nil {
				return err
			}

			_, err = iam.NewRolePolicyAttachment(ctx, "ec2SNSPolicy", &iam.RolePolicyAttachmentArgs{
				Role:      role.ID(),
				PolicyArn: customSNSPolicy.Arn,
			})
			if err != nil {
				return err
			}

			instanceProfile, err := iam.NewInstanceProfile(ctx, "ec2CloudWatchProfile", &iam.InstanceProfileArgs{
				Role: role.Name,
			})
			if err != nil {
				return err
			}

			// Create an EC2 instance
			userData := fmt.Sprintf(`#!/bin/bash
ENV_FILE="/opt/webapp.dev.env"
sudo echo "PORT=%v" >> ${ENV_FILE}
sudo echo "DB_USER=%v" >> ${ENV_FILE}
sudo echo "DB_PASS=%v" >> ${ENV_FILE}
sudo echo "DB_HOST='%v'" >> ${ENV_FILE}
sudo echo "DB_PORT=%v" >> ${ENV_FILE}
sudo echo "DB_NAME=%v" >> ${ENV_FILE}
sudo echo "DRIVER_NAME=%v" >> ${ENV_FILE}
sudo echo "USER_DATA_FILE_PATH='%v'" >> ${ENV_FILE}
sudo echo "MIGRATION_FILE_PATH='%v'" >> ${ENV_FILE}
sudo echo "LOG_FILE_PATH='%v'" >> ${ENV_FILE}
sudo echo "METRIC_SERVER_PORT=%d" >> ${ENV_FILE}
sudo echo "MAILGUN_API_KEY='%v'" >> ${ENV_FILE}
sudo echo "MAILGUN_DOMAIN='%v'" >> ${ENV_FILE}
sudo echo "MAILGUN_SENDER_EMAIL='%v'" >> ${ENV_FILE}
sudo echo "TOPIC_ARN=%v" >> ${ENV_FILE}
sudo chown ec2-user:ec2-group ${ENV_FILE}
sudo chmod 644 ${ENV_FILE}

# Restart Cloud watch agent
sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a fetch-config -m ec2 -c file:/home/ec2-user/webapp/observability-config.json -s
sudo systemctl restart amazon-cloudwatch-agent
`, configData.InboundPorts["customPort"], configData.RDSInstanceMetadata.Username,
				configData.RDSInstanceMetadata.Password, rdsEndpoint, configData.RDSInstanceMetadata.AllowsPort,
				configData.RDSInstanceMetadata.DbName, configData.RDSInstanceMetadata.DbDriver,
				configData.EC2InstanceMetadata.UserDataFilePath, configData.EC2InstanceMetadata.MigrationsFilePath,
				configData.EC2InstanceMetadata.LogFilePath, configData.EC2InstanceMetadata.MetricServerPort,
				configData.MailerClientCreds.APIKey, configData.MailerClientCreds.Domain, configData.MailerClientCreds.Email,
				fmt.Sprintf("arn:aws:sns:%v:%v:%v", configData.ResourceParams.Region, configData.ResourceParams.AccountID, configData.ResourceParams.SNSTopic))

			//webappInstance, err := ec2.NewInstance(ctx, configData.EC2InstanceMetadata.InstanceName, &ec2.InstanceArgs{
			//	InstanceType:             pulumi.String(configData.EC2InstanceMetadata.InstanceType),
			//	AssociatePublicIpAddress: pulumi.Bool(configData.EC2InstanceMetadata.AssociatePublicIpAddress),
			//	KeyName:                  pulumi.String(configData.EC2InstanceMetadata.SSHKeyName),
			//	Ami:                      pulumi.String(configData.EC2InstanceMetadata.AmiID),
			//	SubnetId:                 publicSubnets[0],
			//	UserData:                 pulumi.String(userData),
			//	VpcSecurityGroupIds:      pulumi.StringArray{appSecurityGroup.ID()},
			//	IamInstanceProfile:       instanceProfile.Name,
			//	EbsBlockDevices: ec2.InstanceEbsBlockDeviceArray{
			//		&ec2.InstanceEbsBlockDeviceArgs{
			//			DeviceName:          pulumi.String(configData.EC2InstanceMetadata.DeviceType),
			//			VolumeType:          pulumi.String(configData.EC2InstanceMetadata.VolumeType),        // Use General Purpose SSD (GP2)
			//			VolumeSize:          pulumi.Int(configData.EC2InstanceMetadata.VolumeSize),           // Set root volume size to 25 GB
			//			DeleteOnTermination: pulumi.Bool(configData.EC2InstanceMetadata.DeleteOnTermination), // Root volume is deleted when instance is terminated
			//		},
			//	},
			//	DisableApiTermination: pulumi.Bool(configData.EC2InstanceMetadata.DisableApiTermination), // Protect against accidental termination is set to "No"
			//	Tags: pulumi.StringMap{
			//		"Name": pulumi.String(configData.EC2InstanceMetadata.InstanceName),
			//	},
			//})
			//if err != nil {
			//	return err
			//}

			appLoadBalancer, err := lb.NewLoadBalancer(ctx, "test", &lb.LoadBalancerArgs{
				Name:             pulumi.String("app-load-balancer"),
				Internal:         pulumi.Bool(false),
				LoadBalancerType: pulumi.String("application"),
				SecurityGroups: pulumi.StringArray{
					loadBalancerSecurityGroup.ID(),
				},
				Subnets: publicSubnetsStrs,
				Tags: pulumi.StringMap{
					"Name": pulumi.String("foobar-elb"),
				},
			})
			if err != nil {
				return err
			}

			appLoadBalancerTargetGroup, err := lb.NewTargetGroup(ctx, "test", &lb.TargetGroupArgs{
				Name:       pulumi.String("app-loadbalancer-tg"),
				Port:       pulumi.Int(8080),
				Protocol:   pulumi.String("HTTP"),
				TargetType: pulumi.String("instance"),
				HealthCheck: lb.TargetGroupHealthCheckArgs{
					Enabled:            pulumi.Bool(true),
					Path:               pulumi.String("/healthz"),
					Port:               pulumi.String("traffic-port"),
					Protocol:           pulumi.String("HTTP"),
					HealthyThreshold:   pulumi.Int(2),
					UnhealthyThreshold: pulumi.Int(2),
					Timeout:            pulumi.Int(3),
					Interval:           pulumi.Int(30),
				},
				VpcId: awsVpc.ID(),
			})
			if err != nil {
				return err
			}

			_, err = lb.NewListener(ctx, "frontEndListener", &lb.ListenerArgs{
				LoadBalancerArn: appLoadBalancer.Arn,
				//Port:            pulumi.Int(80),
				//Protocol:        pulumi.String("HTTP"),
				Port:           pulumi.Int(443),
				CertificateArn: pulumi.String(configData.ResourceParams.CertificateArn),
				SslPolicy:      pulumi.String("ELBSecurityPolicy-TLS13-1-2-2021-06"),
				Protocol:       pulumi.String("HTTPS"),
				DefaultActions: lb.ListenerDefaultActionArray{
					&lb.ListenerDefaultActionArgs{
						Type:           pulumi.String("forward"),
						TargetGroupArn: appLoadBalancerTargetGroup.Arn,
					},
				},
			})
			if err != nil {
				return err
			}

			launchTemplate, err := ec2.NewLaunchTemplate(ctx, "example_launch_template", &ec2.LaunchTemplateArgs{
				Name:                  pulumi.String("asg_launch_config"),
				InstanceType:          pulumi.String(configData.EC2InstanceMetadata.InstanceType),
				KeyName:               pulumi.String(configData.EC2InstanceMetadata.SSHKeyName),
				ImageId:               pulumi.String(configData.EC2InstanceMetadata.AmiID),
				IamInstanceProfile:    &ec2.LaunchTemplateIamInstanceProfileArgs{Name: instanceProfile.Name},
				UserData:              pulumi.String(base64.StdEncoding.EncodeToString([]byte(userData))),
				DisableApiTermination: pulumi.Bool(configData.EC2InstanceMetadata.DisableApiTermination),
				Tags: pulumi.StringMap{
					"Name": pulumi.String(configData.EC2InstanceMetadata.InstanceName),
				},
				NetworkInterfaces: ec2.LaunchTemplateNetworkInterfaceArray{
					&ec2.LaunchTemplateNetworkInterfaceArgs{
						AssociatePublicIpAddress: pulumi.String("true"),
						SecurityGroups: pulumi.StringArray{
							appSecurityGroup.ID(),
						},
					},
				},
				BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
					&ec2.LaunchTemplateBlockDeviceMappingArgs{
						DeviceName: pulumi.String(configData.EC2InstanceMetadata.DeviceType),
						Ebs: &ec2.LaunchTemplateBlockDeviceMappingEbsArgs{
							VolumeType:          pulumi.String(configData.EC2InstanceMetadata.VolumeType), // Use General Purpose SSD (GP2)
							VolumeSize:          pulumi.Int(configData.EC2InstanceMetadata.VolumeSize),    // Set root volume size to 25 GB
							DeleteOnTermination: pulumi.String("true"),                                    // Root volume is deleted when instance is terminated
						},
					},
				},
			})
			if err != nil {
				return err
			}

			launchTemplateVersion := launchTemplate.LatestVersion.ApplyT(func(num int) string {
				return strconv.Itoa(num)
			}).(pulumi.StringOutput)

			autoscalingGroup, err := autoscaling.NewGroup(ctx, "example_auto_scaling_group", &autoscaling.GroupArgs{
				Name:               pulumi.String("webapp-auto-scaling-group"),
				VpcZoneIdentifiers: publicSubnetsStrs,
				DefaultCooldown:    pulumi.IntPtr(60),
				DesiredCapacity:    pulumi.IntPtr(1),
				MaxSize:            pulumi.Int(3),
				MinSize:            pulumi.Int(1),
				//HealthCheckType:        pulumi.String("EC2"),
				HealthCheckGracePeriod: pulumi.Int(300),
				Tags: autoscaling.GroupTagArray{
					&autoscaling.GroupTagArgs{
						Key:               pulumi.String("AutoScalingGroup"),
						Value:             pulumi.String("TagProperty"),
						PropagateAtLaunch: pulumi.Bool(true),
					},
				},
				TargetGroupArns: pulumi.StringArray{
					appLoadBalancerTargetGroup.Arn,
				},
				LaunchTemplate: &autoscaling.GroupLaunchTemplateArgs{
					Id:      launchTemplate.ID(),
					Version: launchTemplateVersion,
				},
			})
			if err != nil {
				return err
			}

			scaleUpPolicy, err := autoscaling.NewPolicy(ctx, "scale_up", &autoscaling.PolicyArgs{
				AdjustmentType:       pulumi.String("ChangeInCapacity"),
				ScalingAdjustment:    pulumi.Int(1),
				Cooldown:             pulumi.Int(60),
				AutoscalingGroupName: autoscalingGroup.Name,
				PolicyType:           pulumi.String("SimpleScaling"),
				//EstimatedInstanceWarmup: pulumi.Int(20),
				//StepAdjustments: autoscaling.PolicyStepAdjustmentArray{
				//	&autoscaling.PolicyStepAdjustmentArgs{
				//		MetricIntervalUpperBound: pulumi.String("5"),
				//		ScalingAdjustment:        pulumi.Int(1),
				//	},
				//},
			})
			if err != nil {
				return err
			}

			scaleDownPolicy, err := autoscaling.NewPolicy(ctx, "scale_down", &autoscaling.PolicyArgs{
				AdjustmentType:       pulumi.String("ChangeInCapacity"),
				ScalingAdjustment:    pulumi.Int(-1),
				Cooldown:             pulumi.Int(60),
				AutoscalingGroupName: autoscalingGroup.Name,
				PolicyType:           pulumi.String("SimpleScaling"),
				//EstimatedInstanceWarmup: pulumi.Int(20),
				//StepAdjustments: autoscaling.PolicyStepAdjustmentArray{
				//	&autoscaling.PolicyStepAdjustmentArgs{
				//		MetricIntervalUpperBound: pulumi.String("3"),
				//		ScalingAdjustment:        pulumi.Int(-1),
				//	},
				//},
			})
			if err != nil {
				return err
			}

			_, err = cloudwatch.NewMetricAlarm(ctx, "cpu_utilization_high_alarm", &cloudwatch.MetricAlarmArgs{
				Name:               pulumi.String("cpu-high"),
				ComparisonOperator: pulumi.String("GreaterThanOrEqualToThreshold"),
				EvaluationPeriods:  pulumi.Int(1),
				MetricName:         pulumi.String("CPUUtilization"),
				Namespace:          pulumi.String("AWS/EC2"),
				Period:             pulumi.Int(60),
				Statistic:          pulumi.String("Average"),
				Threshold:          pulumi.Float64(5),
				Dimensions: pulumi.StringMap{
					"AutoScalingGroupName": autoscalingGroup.Name,
				},
				AlarmDescription: pulumi.String("Alarm when CPU exceeds 3% threshold"),
				AlarmActions: pulumi.Array{
					scaleUpPolicy.Arn,
				},
			})

			_, err = cloudwatch.NewMetricAlarm(ctx, "cpu_utilization_low_alarm", &cloudwatch.MetricAlarmArgs{
				Name:               pulumi.String("cpu-low"),
				ComparisonOperator: pulumi.String("LessThanOrEqualToThreshold"),
				EvaluationPeriods:  pulumi.Int(1),
				MetricName:         pulumi.String("CPUUtilization"),
				Namespace:          pulumi.String("AWS/EC2"),
				Period:             pulumi.Int(60),
				Statistic:          pulumi.String("Average"),
				Threshold:          pulumi.Float64(3),
				Dimensions: pulumi.StringMap{
					"AutoScalingGroupName": autoscalingGroup.Name,
				},
				AlarmDescription: pulumi.String("Alarm when CPU falls below 3% threshold"),
				AlarmActions: pulumi.Array{
					scaleDownPolicy.Arn,
				},
			})

			_, err = route53.NewRecord(ctx, configData.Dns.ARecordName, &route53.RecordArgs{
				Name:   pulumi.String(configData.Dns.Domain),
				Type:   pulumi.String(configData.Dns.Type),
				ZoneId: pulumi.String(configData.Dns.HostedZoneID),
				//Records:        pulumi.StringArray{loadBalancer.DnsName},
				//Ttl: pulumi.Int(configData.Dns.Ttl),
				Aliases: route53.RecordAliasArray{
					&route53.RecordAliasArgs{
						Name:                 appLoadBalancer.DnsName,
						ZoneId:               appLoadBalancer.ZoneId,
						EvaluateTargetHealth: pulumi.Bool(true),
					},
				},
				AllowOverwrite: pulumi.BoolPtr(true),
			})
			if err != nil {
				return err
			}

			return nil
		})

		// Export the subnet and route table IDs for later use
		//ctx.Export("LoadBalancerIP", loadBalancer.DnsName)
		//ctx.Export("securityGroupId", securityGroup.ID())
		//ctx.Export("publicSubnetIDs", publicSubnetsStrs)
		//ctx.Export("privateSubnetIDs", privateSubnetsStrs)
		//ctx.Export("rdsEndpoint", rdsInstance.Endpoint.ToStringOutput())
		//ctx.Export("privateRouteTableIDs", pulumi.ToStringArray(pulumi.Map(privateRouteTables, func(rt *ec2.RouteTable) pulumi.IDOutput { return rt.ID() })))

		return nil
	})
}
