package aws

import (
  "context"
  "fmt"
  "github.com/aws/aws-sdk-go-v2/config"
  "github.com/aws/aws-sdk-go-v2/service/autoscaling"
  "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
  "regexp"
  "strings"

  "github.com/aws/aws-sdk-go-v2/service/ec2"
  "time"
)

var (
  targetGroupNameRegExp = regexp.MustCompile("^[a-zA-Z0-9-]+$")
  balancerNameRegExp    = regexp.MustCompile("^[a-zA-Z-]+$")
)

type RunConfig struct {
  InstanceID             string
  GroupName              string
  HealthPath             string
  DaemonPort             int32
  InstancesCount         int32
  HealthCheckGracePeriod time.Duration
  UpdateTimeout          time.Duration
  UpdateTick             time.Duration
}

func (c *RunConfig) GetGroupName() string {
  return c.GroupName
}

func (c *RunConfig) GetLaunchTemplateName() string {
  return c.GroupName
}

func (c *RunConfig) GetTargetGroupName() string {
  return strings.ReplaceAll(c.GroupName, "_", "-")
}

func (c *RunConfig) GetBalancerName() string {
  return strings.ReplaceAll(c.GroupName, "_", "-")
}

func (c *RunConfig) GetAMIName() string {
  return c.GroupName + " v1"
}

func validateELBName(name string, title string) error {
  if len(name) > 32 {
    return fmt.Errorf("%s name will be %q, it shouldn't contain more than 32 symbols, but contains %d", title, name, len(name))
  }
  if !targetGroupNameRegExp.MatchString(name) {
    return fmt.Errorf("%s name will be %q, it must contain only alphanumeric characters or hyphens", title, name)
  }
  if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
    return fmt.Errorf("%s name will be %q, it must not begin or end with a hyphen", title, name)
  }
  if strings.HasPrefix(name, "internal") {
    return fmt.Errorf("%s name will be %q, it must not begin with \"internal-\"", title, name)
  }
  return nil
}

func (c *RunConfig) ValidateArtifactNames() error {
  if err := validateELBName(c.GetBalancerName(), "load balancer"); err != nil {
    return err
  }
  if err := validateELBName(c.GetTargetGroupName(), "target group"); err != nil {
    return err
  }
  return nil
}

type Client struct {
  autoscalingClient *autoscaling.Client
  ec2Client         *ec2.Client
  elbClient         *elasticloadbalancingv2.Client
  rc                *RunConfig
  ctx               context.Context
  region            string
}

func NewClient(ctx context.Context, rc *RunConfig) (*Client, error) {
  awsConfig, err := config.LoadDefaultConfig(ctx)
  if err != nil {
    return nil, fmt.Errorf("cannot load the AWS configuration: %v", err)
  }
  return &Client{
    autoscalingClient: autoscaling.New(autoscaling.Options{
      Credentials: awsConfig.Credentials,
      Region:      awsConfig.Region,
    }),
    ec2Client: ec2.New(ec2.Options{
      Credentials: awsConfig.Credentials,
      Region:      awsConfig.Region,
    }),
    elbClient: elasticloadbalancingv2.New(elasticloadbalancingv2.Options{
      Credentials: awsConfig.Credentials,
      Region:      awsConfig.Region,
    }),
    rc:     rc,
    ctx:    ctx,
    region: awsConfig.Region,
  }, nil
}

func (c *Client) GetAMILink(amiID string) string {
  return fmt.Sprintf("https://console.aws.amazon.com/ec2/v2/home?region=%s#ImageDetails:imageId=%s", c.region, amiID)
}

func (c *Client) GetLaunchTemplateLink(launchTemplateID string) string {
  return fmt.Sprintf("https://console.aws.amazon.com/ec2/v2/home?region=%s#LaunchTemplateDetails:launchTemplateId=%s", c.region, launchTemplateID)
}

func (c *Client) GetTargetGroupLink(targetGroupArn string) string {
  return fmt.Sprintf("https://console.aws.amazon.com/ec2/v2/home?region=%s#TargetGroup:targetGroupArn=%s", c.region, targetGroupArn)
}

func (c *Client) GetLoadBalancerLink() string {
  return fmt.Sprintf("https://console.aws.amazon.com/ec2/v2/home?region=%s#LoadBalancers:search=%s", c.region, c.rc.GetBalancerName())
}

func (c *Client) GetAutoScalingGroupLink() string {
  return fmt.Sprintf("https://console.aws.amazon.com/ec2autoscaling/home?region=%s#/details/%s", c.region, c.rc.GetGroupName())
}
