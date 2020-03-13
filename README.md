# prometheus-rds-sd

Generate [`file_sd`](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#file_sd_config) file of Prometheus for Amazon RDS.

## Usage

```
./prometheus-rds-sd --output.file=/path/to/rds_sd.json --refresh.interval=120
```

## Metadata

The following meta labels are available on targets during relabeling:

- `__meta_rds_availability_zone`: the availability zone in which the instance is running
- `__meta_rds_engine`: the RDS engine name
- `__meta_rds_engine_version`: the RDS engine version
- `__meta_rds_instance_id`: the RDS instance ID
- `__meta_rds_instance_state`: the state of the RDS instance
- `__meta_rds_instance_type`: the type of the RDS instance
- `__meta_rds_tag_<tagkey>`: each tag value of the instance
- `__meta_rds_vpc_id`: the ID of the VPC in which the instance is running

## Output example

```
[
    {
        "targets": [
            "rds-example-001.xxxxxxxxxxxx.ap-northeast-1.rds.amazonaws.com:3306"
        ],
        "labels": {
            "__address__": "rds-example-001.xxxxxxxxxxxx.ap-northeast-1.rds.amazonaws.com:3306",
            "__meta_rds_availability_zone": "ap-northeast-1b",
            "__meta_rds_engine": "aurora",
            "__meta_rds_engine_version": "5.6.mysql_aurora.1.19.2",
            "__meta_rds_instance_id": "rds-example-001",
            "__meta_rds_instance_state": "available",
            "__meta_rds_instance_type": "db.t3.medium",
            "__meta_rds_tag_Environment": "development",
            "__meta_rds_vpc_id": "vpc-xxxxxxxx"
        }
    },
    {
        "targets": [
            "rds-example-002.xxxxxxxxxxxx.ap-northeast-1.rds.amazonaws.com:5432"
        ],
        "labels": {
            "__address__": "rds-example-002.xxxxxxxxxxxx.ap-northeast-1.rds.amazonaws.com:5432",
            "__meta_rds_availability_zone": "ap-northeast-1c",
            "__meta_rds_engine": "postgres",
            "__meta_rds_engine_version": "9.6.11",
            "__meta_rds_instance_id": "rds-example-002",
            "__meta_rds_instance_state": "available",
            "__meta_rds_instance_type": "db.t2.small",
            "__meta_rds_tag_Environment": "production",
            "__meta_rds_vpc_id": "vpc-xxxxxxxx""
        }
    },
]
```

