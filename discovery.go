package main

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/util/strutil"
)

const (
	rdsLabel              = model.MetaLabelPrefix + "rds_"
	rdsLabelAZ            = rdsLabel + "availability_zone"
	rdsLabelInstanceID    = rdsLabel + "instance_id"
	rdsLabelInstanceState = rdsLabel + "instance_state"
	rdsLabelInstanceType  = rdsLabel + "instance_type"
	rdsLabelEngine        = rdsLabel + "engine"
	rdsLabelEngineVersion = rdsLabel + "engine_version"
	rdsLabelTag           = rdsLabel + "tag_"
	rdsLabelVPCID         = rdsLabel + "vpc_id"
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
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		var tgs []*targetgroup.Group

		sess := session.Must(session.NewSession())
		client := rds.New(sess)

		input := &rds.DescribeDBInstancesInput{
			Filters: d.filters,
		}

		if err := client.DescribeDBInstancesPagesWithContext(ctx, input, func(out *rds.DescribeDBInstancesOutput, lastPage bool) bool {
			for _, dbi := range out.DBInstances {
				labels := model.LabelSet{
					rdsLabelInstanceID: model.LabelValue(*dbi.DBInstanceIdentifier),
				}

				labels[rdsLabelAZ] = model.LabelValue(*dbi.AvailabilityZone)
				labels[rdsLabelInstanceState] = model.LabelValue(*dbi.DBInstanceStatus)
				labels[rdsLabelInstanceType] = model.LabelValue(*dbi.DBInstanceClass)

				addr := net.JoinHostPort(*dbi.Endpoint.Address, strconv.FormatInt(*dbi.Endpoint.Port, 10))
				labels[model.AddressLabel] = model.LabelValue(addr)

				labels[rdsLabelEngine] = model.LabelValue(*dbi.Engine)
				labels[rdsLabelEngineVersion] = model.LabelValue(*dbi.EngineVersion)

				labels[rdsLabelVPCID] = model.LabelValue(*dbi.DBSubnetGroup.VpcId)

				tags, err := listTagsForInstance(client, dbi)
				if err != nil {
					level.Error(d.logger).Log("msg", "could not list tags for db instance", "err", err)
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
			return true
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
