package aws

import (
  "context"
  "fmt"
  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/config"
  "github.com/aws/aws-sdk-go-v2/service/autoscaling"
  "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
  "log"
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
  if len(name) < 3 {
    return fmt.Errorf("%s name will be %q, it shouldn't contain less than 3 symbols, but contains %d", title, name, len(name))
  }
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
  amiID                           string
  launchTemplateID                string
  targetGroupARN                  string
  loadBalancerName                string
  loadBalancerDNSName             string
  loadBalancerARN                 string
  autoScalingGroupCreationStarted bool

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

func (c *Client) ReportCreatedArtifacts() {
  log.Printf("AMI link: %s", c.GetAMILink(c.amiID))
  log.Printf("Launch template link: %s", c.GetLaunchTemplateLink(c.launchTemplateID))
  log.Printf("Target group link: %s", c.GetTargetGroupLink(c.targetGroupARN))
  log.Printf("Balancer link: %s", c.GetLoadBalancerLink())
  log.Printf("Auto Scalingr group link: %s", c.GetAutoScalingGroupLink())
  log.Printf("check out the health status: http://%s:%d%s", c.loadBalancerDNSName, c.rc.DaemonPort, c.rc.HealthPath)
}

func (c *Client) Cleanup() {
  var errorMessages []string
  if c.amiID != "" {
    _, err := c.ec2Client.DeregisterImage(c.ctx, &ec2.DeregisterImageInput{
      ImageId: aws.String(c.amiID),
    })
    if err != nil {
      errorMessages = append(errorMessages, fmt.Sprintf("cannot deregister %q: %v", c.amiID, err))
    } else {
      log.Printf("deregistered %q", c.amiID)
    }
  }
  if c.launchTemplateID != "" {
    _, err := c.ec2Client.DeleteLaunchTemplate(c.ctx, &ec2.DeleteLaunchTemplateInput{
      LaunchTemplateId: aws.String(c.launchTemplateID),
    })
    if err != nil {
      errorMessages = append(errorMessages, fmt.Sprintf("cannot delete launch template %q: %v", c.launchTemplateID, err))
    } else {
      log.Printf("deleted launch template %q", c.launchTemplateID)
    }
  }
  if c.targetGroupARN != "" {
    _, err := c.elbClient.DeleteTargetGroup(c.ctx, &elasticloadbalancingv2.DeleteTargetGroupInput{
      TargetGroupArn: aws.String(c.targetGroupARN),
    })
    if err != nil {
      errorMessages = append(errorMessages, fmt.Sprintf("cannot delete target group %q: %v", c.targetGroupARN, err))
    } else {
      log.Printf("deleted target group %q", c.targetGroupARN)
    }
  }
  if c.loadBalancerARN != "" {
    _, err := c.elbClient.DeleteLoadBalancer(c.ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
      LoadBalancerArn: aws.String(c.loadBalancerARN),
    })
    if err != nil {
      errorMessages = append(errorMessages, fmt.Sprintf("cannot delete load balancer %q: %v", c.loadBalancerName, err))
    } else {
      log.Printf("deleted load balancer %q", c.loadBalancerName)
    }
  }
  if c.autoScalingGroupCreationStarted {
    _, err := c.autoscalingClient.DeleteAutoScalingGroup(c.ctx, &autoscaling.DeleteAutoScalingGroupInput{
      AutoScalingGroupName: aws.String(c.rc.GetGroupName()),
      ForceDelete:          aws.Bool(true),
    })
    if err != nil {
      errorMessages = append(errorMessages, fmt.Sprintf("cannot delete the auto scaling group %q: %v", c.rc.GetGroupName(), err))
    } else {
      log.Printf("deleted auto scaling group %q", c.rc.GetGroupName())
    }
  }
  for _, errorMessage := range errorMessages {
    log.Println(errorMessage)
  }
}
