package aws

import (
  "encoding/json"
  "fmt"
  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/service/ec2"
  "github.com/aws/aws-sdk-go-v2/service/ec2/types"
  "log"
)

func extractLicenseSpecifications(instanceData *types.Instance) ([]types.LaunchTemplateLicenseConfigurationRequest, error) {
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

func extractPlacement(instanceData *types.Instance) (*types.LaunchTemplatePlacementRequest, error) {
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

func (c *Client) generateLaunchTemplateData(amiID string, instanceData *types.Instance) (*types.RequestLaunchTemplateData, error) {
  licenseSpecificationsRequests, err := extractLicenseSpecifications(instanceData)
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
    Placement:             placementRequest,
  }, nil
}

func (c *Client) CreateLaunchTemplate(instanceData *types.Instance) error {
  launchTemplateData, err := c.generateLaunchTemplateData(c.amiID, instanceData)
  if err != nil {
    return fmt.Errorf("cannot generate launch template data from ami %s: %v", c.amiID, err)
  }
  res, err := c.ec2Client.CreateLaunchTemplate(c.ctx, &ec2.CreateLaunchTemplateInput{
    LaunchTemplateData: launchTemplateData,
    LaunchTemplateName: aws.String(c.rc.GetLaunchTemplateName()),
  })
  if err != nil {
    return fmt.Errorf("cannot create launch template from ami %s: %v", c.amiID, err)
  }
  log.Printf("created launch template %q", c.rc.GroupName)
  c.launchTemplateID = *res.LaunchTemplate.LaunchTemplateId
  return nil
}
