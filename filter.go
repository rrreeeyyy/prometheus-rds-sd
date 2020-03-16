package main

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"gopkg.in/alecthomas/kingpin.v2"
)

type rdsFiltersValue []*rds.Filter

func (r *rdsFiltersValue) Set(value string) error {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected Name=NAME,Values=VALUES got '%s'", value)
	}

	nargs := strings.SplitN(parts[0], "=", 2)
	if len(nargs) != 2 && nargs[0] != "Name" {
		return fmt.Errorf("expected Name=NAME,Values=VALUES got '%s'", value)
	}

	vargs := strings.SplitN(parts[1], "=", 2)
	if len(vargs) != 2 && vargs[0] != "Values" {
		return fmt.Errorf("expected Name=NAME,Values=VALUES got '%s'", value)
	}
	values := strings.Split(vargs[1], ",")

	filter := rds.Filter{
		Name:   aws.String(nargs[1]),
		Values: make([]*string, len(values)),
	}

	for i, v := range values {
		filter.Values[i] = aws.String(v)
	}

	*r = append(*r, &filter)

	return nil
}

func (r *rdsFiltersValue) String() string {
	return ""
}

func (r *rdsFiltersValue) IsCumulative() bool {
	return true
}

func rdsFilters(s kingpin.Settings) (target *[]*rds.Filter) {
	target = &[]*rds.Filter{}
	s.SetValue((*rdsFiltersValue)(target))
	return
}
