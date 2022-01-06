package aws

import (
  "fmt"
  "github.com/aws/aws-sdk-go-v2/service/ec2"
  "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func (c *Client) DescribeInstance() (*types.Instance, error) {
  describedInstances, err := c.ec2Client.DescribeInstances(c.ctx, &ec2.DescribeInstancesInput{
    InstanceIds: []string{c.rc.InstanceID},
  })
  if err != nil {
    return nil, fmt.Errorf("cannot receive description for the instance id %s", c.rc.InstanceID)
  }
  if len(describedInstances.Reservations) != 1 {
    return nil, fmt.Errorf("received wrong number %d != 1 of reservations for instance id %s", len(describedInstances.Reservations), c.rc.InstanceID)
  }
  if len(describedInstances.Reservations[0].Instances) != 1 {
    return nil, fmt.Errorf("received wrong number %d != 1 of instances for instance id %s", len(describedInstances.Reservations[0].Instances), c.rc.InstanceID)
  }
  instanceData := describedInstances.Reservations[0].Instances[0]
  return &instanceData, nil
}
