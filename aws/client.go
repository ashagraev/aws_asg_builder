package aws

import (
  "context"
  "fmt"
  "github.com/aws/aws-sdk-go-v2/config"
  "github.com/aws/aws-sdk-go-v2/service/autoscaling"
  "github.com/aws/aws-sdk-go-v2/service/ec2"
  "time"
)

type RunConfig struct {
  InstanceID             string
  GroupName              string
  HealthPath             string
  DaemonPort             int
  InstancesCount         int32
  HealthCheckGracePeriod time.Duration
  UpdateTimeout          time.Duration
  UpdateTick             time.Duration
}

type Client struct {
  autoscalingClient *autoscaling.Client
  ec2Client         *ec2.Client
  rc                *RunConfig
  ctx               context.Context
}

func NewClient(ctx context.Context, rc *RunConfig) (*Client, error) {
  awsConfig, err := config.LoadDefaultConfig(ctx)
  if err != nil {
    return nil, fmt.Errorf("cannot load the AWS configuration: %v", err)
  }
  return &Client{
    autoscalingClient: autoscaling.New(autoscaling.Options{
      Region:      awsConfig.Region,
      Credentials: awsConfig.Credentials,
    }),
    ec2Client: ec2.New(ec2.Options{
      Region:      awsConfig.Region,
      Credentials: awsConfig.Credentials,
    }),
    rc:  rc,
    ctx: ctx,
  }, nil
}
