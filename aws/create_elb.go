package aws

import (
  "fmt"
  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/service/ec2"
  ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
  "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
  "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
  "log"
  "strings"
  "time"
)

func (c *Client) CreateTargetGroup(instanceDescription *ec2types.Instance) error {
  targetGroupName := strings.ReplaceAll(c.rc.GroupName, "_", "-")
  res, err := c.elbClient.CreateTargetGroup(c.ctx, &elasticloadbalancingv2.CreateTargetGroupInput{
    Name:               aws.String(c.rc.GetTargetGroupName()),
    HealthCheckEnabled: aws.Bool(true),
    HealthCheckPath:    aws.String(c.rc.HealthPath),
    HealthCheckPort:    aws.String(fmt.Sprintf("%d", c.rc.DaemonPort)),
    Protocol:           types.ProtocolEnumHttp,
    VpcId:              instanceDescription.VpcId,
    Port:               aws.Int32(c.rc.DaemonPort),
  })
  if err != nil {
    return fmt.Errorf("cannot register the target group %q: %v", targetGroupName, err)
  }
  if len(res.TargetGroups) != 1 {
    return fmt.Errorf("created wrong %d != 1 number of target groups with name %q", len(res.TargetGroups), targetGroupName)
  }
  log.Printf("created target group %q (%s)", targetGroupName, *res.TargetGroups[0].TargetGroupArn)
  c.targetGroupARN = *res.TargetGroups[0].TargetGroupArn
  return nil
}

func (c *Client) GetDefaultVPCID() (string, error) {
  var nextToken *string
  for {
    vpcRes, err := c.ec2Client.DescribeVpcs(c.ctx, &ec2.DescribeVpcsInput{NextToken: nextToken})
    if err != nil {
      return "", fmt.Errorf("cannot describe VPCs: %v", err)
    }
    for _, vpc := range vpcRes.Vpcs {
      if *vpc.IsDefault && vpc.State != ec2types.VpcStateAvailable {
        return "", fmt.Errorf("the default VPC %s is not in the available state %q", *vpc.VpcId, vpc.State)
      }
      if *vpc.IsDefault {
        return *vpc.VpcId, nil
      }
    }
    nextToken = vpcRes.NextToken
    if nextToken == nil {
      break
    }
  }
  return "", nil
}

func (c *Client) GetSubnets(defaultVPCID string) ([]string, error) {
  var nextToken *string
  var subnetIDs []string
  for {
    subnets, err := c.ec2Client.DescribeSubnets(c.ctx, &ec2.DescribeSubnetsInput{
      NextToken: nextToken,
    })
    if err != nil {
      return nil, fmt.Errorf("cannot read the list of subnets: %v", err)
    }
    for _, s := range subnets.Subnets {
      if *s.VpcId == defaultVPCID && *s.DefaultForAz {
        subnetIDs = append(subnetIDs, *s.SubnetId)
      }
    }
    if subnets.NextToken == nil {
      break
    }
    nextToken = subnets.NextToken
  }
  return subnetIDs, nil
}

func (c *Client) CreateLoadBalancer(subnetIDs []string) error {
  balancerName := strings.ReplaceAll(c.rc.GroupName, "_", "-")
  createLoadBalancerRes, err := c.elbClient.CreateLoadBalancer(c.ctx, &elasticloadbalancingv2.CreateLoadBalancerInput{
    Name:    aws.String(c.rc.GetBalancerName()),
    Scheme:  types.LoadBalancerSchemeEnumInternetFacing,
    Type:    types.LoadBalancerTypeEnumApplication,
    Subnets: subnetIDs,
  })
  if err != nil {
    return fmt.Errorf("cannot create an elastic load balancer %q: %v", balancerName, err)
  }
  if len(createLoadBalancerRes.LoadBalancers) != 1 {
    return fmt.Errorf("created wrong %d != 1 number of load balancers with name %q", len(createLoadBalancerRes.LoadBalancers), balancerName)
  }
  finishTime := time.Now().Add(c.rc.UpdateTimeout)
  for time.Now().Before(finishTime) {
    describeLoadBalancersRes, err := c.elbClient.DescribeLoadBalancers(c.ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
      LoadBalancerArns: []string{*createLoadBalancerRes.LoadBalancers[0].LoadBalancerArn},
    })
    if err != nil {
      log.Printf("cannot get description of the balancer %q: %v", balancerName, err)
      time.Sleep(c.rc.UpdateTick)
      continue
    }
    if len(describeLoadBalancersRes.LoadBalancers) != 1 {
      return fmt.Errorf("received wrong %d != 1 number of load balancers with arn %s", len(describeLoadBalancersRes.LoadBalancers), *createLoadBalancerRes.LoadBalancers[0].LoadBalancerArn)
    }
    state := describeLoadBalancersRes.LoadBalancers[0].State.Code
    log.Printf("load balancer %q: %s", balancerName, state)
    if state != types.LoadBalancerStateEnumProvisioning {
      if state != types.LoadBalancerStateEnumActive {
        return fmt.Errorf("load balancer ended up not in an active status %q", state)
      }
      _, err := c.elbClient.CreateListener(c.ctx, &elasticloadbalancingv2.CreateListenerInput{
        DefaultActions: []types.Action{
          {
            Type: types.ActionTypeEnumForward,
            ForwardConfig: &types.ForwardActionConfig{
              TargetGroups: []types.TargetGroupTuple{
                {
                  TargetGroupArn: aws.String(c.targetGroupARN),
                  Weight:         aws.Int32(1),
                },
              },
            },
          },
        },
        LoadBalancerArn: createLoadBalancerRes.LoadBalancers[0].LoadBalancerArn,
        Port:            aws.Int32(c.rc.DaemonPort),
        Protocol:        types.ProtocolEnumHttp,
      })
      if err != nil {
        return fmt.Errorf("cannot crate listener for the load balancer %q: %v", balancerName, err)
      }
      c.loadBalancerDNSName = *describeLoadBalancersRes.LoadBalancers[0].DNSName
      return nil
    }
    time.Sleep(c.rc.UpdateTick)
  }
  return fmt.Errorf("the balancer %q has not become ready within the timeout %v", balancerName, c.rc.UpdateTimeout)
}
