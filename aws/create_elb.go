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

func (c *Client) CreateTargetGroup(instanceDescription *ec2types.Instance) (string, error) {
  targetGroupName := strings.ReplaceAll(c.rc.GroupName, "_", "-")
  res, err := c.elbClient.CreateTargetGroup(c.ctx, &elasticloadbalancingv2.CreateTargetGroupInput{
    Name:               aws.String(targetGroupName),
    HealthCheckEnabled: aws.Bool(true),
    HealthCheckPath:    aws.String(c.rc.HealthPath),
    HealthCheckPort:    aws.String(fmt.Sprintf("%d", c.rc.DaemonPort)),
    Protocol:           types.ProtocolEnumHttp,
    VpcId:              instanceDescription.VpcId,
    Port:               aws.Int32(c.rc.DaemonPort),
  })
  if err != nil {
    return "", fmt.Errorf("cannot register the target group %q: %v", targetGroupName, err)
  }
  if len(res.TargetGroups) != 1 {
    return "", fmt.Errorf("created wrong %d != 1 number of target groups with name %q", len(res.TargetGroups), targetGroupName)
  }
  log.Printf("created target group %q (%s)", targetGroupName, *res.TargetGroups[0].TargetGroupArn)
  return *res.TargetGroups[0].TargetGroupArn, nil
}

func (c *Client) GetSubnets() ([]string, error) {
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
      // TODO: better subnets exploration method.
      if *s.DefaultForAz {
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

func (c *Client) CreateLoadBalancer(targetGroupArn string, subnetIDs []string) (*types.LoadBalancer, error) {
  balancerName := strings.ReplaceAll(c.rc.GroupName, "_", "-")
  createLoadBalancerRes, err := c.elbClient.CreateLoadBalancer(c.ctx, &elasticloadbalancingv2.CreateLoadBalancerInput{
    Name:    aws.String(balancerName),
    Scheme:  types.LoadBalancerSchemeEnumInternetFacing,
    Type:    types.LoadBalancerTypeEnumApplication,
    Subnets: subnetIDs,
  })
  if err != nil {
    return nil, fmt.Errorf("cannot create an elastic load balancer %q: %v", balancerName, err)
  }
  if len(createLoadBalancerRes.LoadBalancers) != 1 {
    return nil, fmt.Errorf("created wrong %d != 1 number of load balancers with name %q", len(createLoadBalancerRes.LoadBalancers), balancerName)
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
      return nil, fmt.Errorf("received wrong %d != 1 number of load balancers with arn %s", len(describeLoadBalancersRes.LoadBalancers), *createLoadBalancerRes.LoadBalancers[0].LoadBalancerArn)
    }
    state := describeLoadBalancersRes.LoadBalancers[0].State.Code
    log.Printf("load balancer %q: %s", balancerName, state)
    if state != types.LoadBalancerStateEnumProvisioning {
      if state != types.LoadBalancerStateEnumActive {
        return nil, fmt.Errorf("load balancer ended up not in an active status %q", state)
      }
      _, err := c.elbClient.CreateListener(c.ctx, &elasticloadbalancingv2.CreateListenerInput{
        DefaultActions: []types.Action{
          {
            Type: types.ActionTypeEnumForward,
            ForwardConfig: &types.ForwardActionConfig{
              TargetGroups: []types.TargetGroupTuple{
                {
                  TargetGroupArn: aws.String(targetGroupArn),
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
        return nil, fmt.Errorf("cannot crate listener for the load balancer %q: %v", balancerName, err)
      }
      return &describeLoadBalancersRes.LoadBalancers[0], nil
    }
    time.Sleep(c.rc.UpdateTick)
  }
  return nil, fmt.Errorf("the balancer %q has not become ready within the timeout %v", balancerName, c.rc.UpdateTimeout)
}
