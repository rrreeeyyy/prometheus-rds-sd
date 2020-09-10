package main

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/util/strutil"
)

const (
	rdsLabel                = model.MetaLabelPrefix + "rds_"
	rdsLabelAZ              = rdsLabel + "availability_zone"
	rdsLabelInstanceID      = rdsLabel + "instance_id"
	rdsLabelResourceID      = rdsLabel + "resource_id"
	rdsLabelInstanceState   = rdsLabel + "instance_state"
	rdsLabelInstanceType    = rdsLabel + "instance_type"
	rdsLabelEngine          = rdsLabel + "engine"
	rdsLabelEngineVersion   = rdsLabel + "engine_version"
	rdsLabelTag             = rdsLabel + "tag_"
	rdsLabelVPCID           = rdsLabel + "vpc_id"
	rdsLabelEndpointAddress = rdsLabel + "endpoint_address"
	rdsLabelEndpointPort    = rdsLabel + "endpoint_port"
)

type discovery struct {
	refreshInterval int
	logger          log.Logger
	filters         []*rds.Filter
}

func newDiscovery(conf sdConfig, logger log.Logger) (*discovery, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	d := &discovery{
		logger:          logger,
		refreshInterval: conf.RefreshInterval,
		filters:         conf.Filters,
	}

	return d, nil
}

func (d *discovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	var region string
	for region == "" {
		var err error
		region, err = getDefaultRegion()
		if err != nil {
			level.Error(d.logger).Log("msg", "could not get default region", "err", err)
			time.Sleep(time.Duration(d.refreshInterval) * time.Second)
			continue
		}
	}
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		var tgs []*targetgroup.Group

		sess := session.Must(session.NewSession())
		client := rds.New(sess, &aws.Config{Region: aws.String(region)})

		input := &rds.DescribeDBInstancesInput{
			Filters: d.filters,
		}

		if err := client.DescribeDBInstancesPagesWithContext(ctx, input, func(out *rds.DescribeDBInstancesOutput, lastPage bool) bool {
			for _, dbi := range out.DBInstances {
				if dbi.Endpoint == nil || dbi.Endpoint.Address == nil || dbi.Endpoint.Port == nil {
					continue // instance is not ready
				}

				labels := model.LabelSet{
					rdsLabelInstanceID: model.LabelValue(*dbi.DBInstanceIdentifier),
				}

				labels[rdsLabelResourceID] = model.LabelValue(*dbi.DbiResourceId)
				labels[rdsLabelAZ] = model.LabelValue(*dbi.AvailabilityZone)
				labels[rdsLabelInstanceState] = model.LabelValue(*dbi.DBInstanceStatus)
				labels[rdsLabelInstanceType] = model.LabelValue(*dbi.DBInstanceClass)

				addr := net.JoinHostPort(*dbi.Endpoint.Address, strconv.FormatInt(*dbi.Endpoint.Port, 10))
				labels[model.AddressLabel] = model.LabelValue(addr)

				labels[rdsLabelEngine] = model.LabelValue(*dbi.Engine)
				labels[rdsLabelEngineVersion] = model.LabelValue(*dbi.EngineVersion)

				labels[rdsLabelVPCID] = model.LabelValue(*dbi.DBSubnetGroup.VpcId)

				labels[rdsLabelEndpointAddress] = model.LabelValue(*dbi.Endpoint.Address)
				labels[rdsLabelEndpointPort] = model.LabelValue(strconv.FormatInt(*dbi.Endpoint.Port, 10))

				tags, err := listTagsForInstance(client, dbi)
				if err != nil {
					level.Error(d.logger).Log("msg", "could not list tags for db instance", "err", err)
					continue
				}

				for _, t := range tags.TagList {
					if t == nil || t.Key == nil || t.Value == nil {
						continue
					}

					name := strutil.SanitizeLabelName(*t.Key)
					labels[rdsLabelTag+model.LabelName(name)] = model.LabelValue(*t.Value)
				}

				tgs = append(tgs, &targetgroup.Group{
					Source:  *dbi.DBInstanceIdentifier,
					Targets: []model.LabelSet{{model.AddressLabel: labels[model.AddressLabel]}},
					Labels:  labels,
				})
			}
			return !lastPage
		}); err != nil {
			level.Error(d.logger).Log("msg", "could not describe db instance", "err", err)
			time.Sleep(time.Duration(d.refreshInterval) * time.Second)
			continue
		}

		ch <- tgs

		select {
		case <-c:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func listTagsForInstance(client *rds.RDS, dbi *rds.DBInstance) (*rds.ListTagsForResourceOutput, error) {
	input := &rds.ListTagsForResourceInput{
		ResourceName: aws.String(*dbi.DBInstanceArn),
	}
	return client.ListTagsForResource(input)
}

func getDefaultRegion() (string, error) {
	var region string

	sess := session.Must(session.NewSession())
	metadata := ec2metadata.New(sess, &aws.Config{
		MaxRetries: aws.Int(0),
	})
	if metadata.Available() {
		var err error
		region, err = metadata.Region()
		if err != nil {
			return "", err
		}
	} else {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
	}

	return region, nil
}
