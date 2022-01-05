package aws

import (
  "encoding/json"
  "fmt"
  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/service/ec2"
  "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func extractNetworkInterfaces(instanceData types.Instance) ([]types.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest, error) {
  networkInterfacesJson, err := json.Marshal(instanceData.NetworkInterfaces)
  if err != nil {
    return nil, fmt.Errorf("cannot marshal network interfaces description of the instance with id %s: %v", *instanceData.InstanceId, err)
  }
  var networkInterfacesArray []interface{}
  if err := json.Unmarshal(networkInterfacesJson, &networkInterfacesArray); err != nil {
    return nil, fmt.Errorf("cannot unmarshal network interfaces description of the instance with id %s: %v", *instanceData.InstanceId, err)
  }
  for idx := range networkInterfacesArray {
    delete(networkInterfacesArray[idx].(map[string]interface{}), "Groups")
  }
  cleanNetworkInterfacesJson, err := json.Marshal(networkInterfacesArray)
  if err != nil {
    return nil, fmt.Errorf("cannot marshal network interfaces description of the instance with id %s: %v", *instanceData.InstanceId, err)
  }
  var networkInterfacesRequests []types.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest
  if err := json.Unmarshal(cleanNetworkInterfacesJson, &networkInterfacesRequests); err != nil {
    return nil, fmt.Errorf("cannot unmarshal network interfaces description of the instance with id %s: %v", *instanceData.InstanceId, err)
  }
  return networkInterfacesRequests, nil
}

func extractLicenseSpecifications(instanceData types.Instance) ([]types.LaunchTemplateLicenseConfigurationRequest, error) {
  licenseSpecificationsJson, err := json.Marshal(instanceData.Licenses)
  if err != nil {
    return nil, fmt.Errorf("cannot marshal license specifications of the instance with id %s: %v", *instanceData.InstanceId, err)
  }
  var licenseSpecificationsRequests []types.LaunchTemplateLicenseConfigurationRequest
  if err := json.Unmarshal(licenseSpecificationsJson, &licenseSpecificationsRequests); err != nil {
    return nil, fmt.Errorf("cannot unmarshal license specifications of the instance with id %s: %v", *instanceData.InstanceId, err)
  }
  return licenseSpecificationsRequests, nil
}

func extractPlacement(instanceData types.Instance) (*types.LaunchTemplatePlacementRequest, error) {
  placementJson, err := json.Marshal(instanceData.Placement)
  if err != nil {
    return nil, fmt.Errorf("cannot marshal placement of the instance with id %s: %v", *instanceData.InstanceId, err)
  }
  placementRequest := &types.LaunchTemplatePlacementRequest{}
  if err := json.Unmarshal(placementJson, &placementRequest); err != nil {
    return nil, fmt.Errorf("cannot unmarshal placement of the instance with id %s: %v", *instanceData.InstanceId, err)
  }
  return placementRequest, nil
}

func (c *Client) generateLaunchTemplateData(amiID string) (*types.RequestLaunchTemplateData, error) {
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
  licenseSpecificationsRequests, err := extractLicenseSpecifications(instanceData)
  if err != nil {
    return nil, err
  }
  networkInterfacesRequests, err := extractNetworkInterfaces(instanceData)
  if err != nil {
    return nil, err
  }
  placementRequest, err := extractPlacement(instanceData)
  if err != nil {
    return nil, err
  }
  return &types.RequestLaunchTemplateData{
    ImageId: aws.String(amiID),
    InstanceMarketOptions: &types.LaunchTemplateInstanceMarketOptionsRequest{
      MarketType: "spot",
    },
    InstanceType:          instanceData.InstanceType,
    KernelId:              instanceData.KernelId,
    KeyName:               instanceData.KeyName,
    LicenseSpecifications: licenseSpecificationsRequests,
    NetworkInterfaces:     networkInterfacesRequests,
    Placement:             placementRequest,
  }, nil
}

func (c *Client) CreateLaunchTemplate(amiID string) (string, error) {
  launchTemplateData, err := c.generateLaunchTemplateData(amiID)
  if err != nil {
    return "", fmt.Errorf("cannot generate launch template data from ami %s: %v", amiID, err)
  }
  res, err := c.ec2Client.CreateLaunchTemplate(c.ctx, &ec2.CreateLaunchTemplateInput{
    LaunchTemplateData: launchTemplateData,
    LaunchTemplateName: aws.String(c.rc.GroupName),
  })
  if err != nil {
    return "", fmt.Errorf("cannot create launch template from ami %s: %v", amiID, err)
  }
  return *res.LaunchTemplate.LaunchTemplateId, nil
}
