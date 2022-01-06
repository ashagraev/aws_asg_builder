package aws

import (
  "fmt"
  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/service/autoscaling"
  "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
  ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
  "log"
  "strings"
  "time"
)

func (c *Client) CreateAutoScalingGroup(launchTemplateID string, instanceData *ec2types.Instance) error {
  targetGroupARN, err := c.CreateTargetGroup(instanceData)
  if err != nil {
    return fmt.Errorf("cannot create a target group: %v", err)
  }
  subnetIDs, err := c.GetSubnets()
  if err != nil {
    return fmt.Errorf("cannot get the list of subnet ids: %v", err)
  }
  log.Printf("load balancer subnets: %s", strings.Join(subnetIDs, ", "))
  loadBalancer, err := c.CreateLoadBalancer(targetGroupARN, subnetIDs)
  if err != nil {
    return fmt.Errorf("cannot create a balancer: %v", err)
  }
  _, err = c.autoscalingClient.CreateAutoScalingGroup(c.ctx, &autoscaling.CreateAutoScalingGroupInput{
    AutoScalingGroupName:   aws.String(c.rc.GroupName),
    MaxSize:                aws.Int32(2 * c.rc.InstancesCount),
    MinSize:                aws.Int32(c.rc.InstancesCount),
    CapacityRebalance:      aws.Bool(true),
    DesiredCapacity:        aws.Int32(c.rc.InstancesCount),
    HealthCheckGracePeriod: aws.Int32(int32(c.rc.HealthCheckGracePeriod.Seconds())),
    HealthCheckType:        aws.String("ELB"),
    LaunchTemplate: &types.LaunchTemplateSpecification{
      LaunchTemplateId: aws.String(launchTemplateID),
    },
    TargetGroupARNs:   []string{targetGroupARN},
    VPCZoneIdentifier: aws.String(strings.Join(subnetIDs, ",")),
  })
  if err != nil {
    return fmt.Errorf("cannot create an autoscaling group: %v", err)
  }
  finishTime := time.Now().Add(c.rc.UpdateTimeout)
  for time.Now().Before(finishTime) {
    res, err := c.autoscalingClient.DescribeAutoScalingGroups(c.ctx, &autoscaling.DescribeAutoScalingGroupsInput{
      AutoScalingGroupNames: []string{c.rc.GroupName},
    })
    if err != nil {
      log.Printf("cannot get description of the autoscaling group %q: %v", c.rc.GroupName, err)
      time.Sleep(c.rc.UpdateTick)
      continue
    }
    if len(res.AutoScalingGroups) != 1 {
      return fmt.Errorf("received wrong %d != 1 number of auto scaling groups with name %q", len(res.AutoScalingGroups), c.rc.GroupName)
    }
    numHealthy := int32(0)
    for _, instance := range res.AutoScalingGroups[0].Instances {
      log.Printf("group %q, instance %q: %s, %s", c.rc.GroupName, *instance.InstanceId, instance.LifecycleState, *instance.HealthStatus)
      if instance.LifecycleState == types.LifecycleStateInService && *instance.HealthStatus == "Healthy" {
        numHealthy++
      }
    }
    log.Printf("group %q: %d instances in total, %d instances are in service and healthy (%d needed)", c.rc.GroupName, len(res.AutoScalingGroups[0].Instances), numHealthy, c.rc.InstancesCount)
    if numHealthy >= c.rc.InstancesCount {
      log.Printf("successfully created an auto scaling group %q", c.rc.GroupName)
      log.Printf("check out the health status: http://%s:%d%s", *loadBalancer.DNSName, c.rc.DaemonPort, c.rc.HealthPath)
      return nil
    }
    time.Sleep(c.rc.UpdateTick)
  }
  return fmt.Errorf("the autoscaling group %s has not become ready within the timeout %v", c.rc.GroupName, c.rc.UpdateTimeout)
}
