package aws

import (
  "fmt"
  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/service/ec2"
  "github.com/aws/aws-sdk-go-v2/service/ec2/types"
  "log"
  "time"
)

func (c *Client) getImageState(imageID string) (types.ImageState, error) {
  imagesDescription, err := c.ec2Client.DescribeImages(c.ctx, &ec2.DescribeImagesInput{
    ImageIds: []string{
      imageID,
    },
  })
  if err != nil {
    return "", fmt.Errorf("cannot get image description for AMI %s: %v", imageID, err)
  }
  if len(imagesDescription.Images) != 1 {
    return "", fmt.Errorf("got %d != 1 images for image ID %s", len(imagesDescription.Images), imageID)
  }
  return imagesDescription.Images[0].State, nil
}

func (c *Client) CreateAMI() error {
  createImageOutput, err := c.ec2Client.CreateImage(c.ctx, &ec2.CreateImageInput{
    InstanceId: aws.String(c.rc.InstanceID),
    Name:       aws.String(c.rc.GetAMIName()),
    NoReboot:   aws.Bool(false),
  })
  if err != nil {
    return fmt.Errorf("cannot create an AMI from instance %s: %v", c.rc.InstanceID, err)
  }
  finishTime := time.Now().Add(c.rc.UpdateTimeout)
  for time.Now().Before(finishTime) {
    imageState, err := c.getImageState(*createImageOutput.ImageId)
    if err != nil {
      log.Printf("cannot get image state: %v", err)
      time.Sleep(c.rc.UpdateTick)
      continue
    }
    log.Printf("%s (%q): %s", *createImageOutput.ImageId, c.rc.GetAMIName(), imageState)
    if imageState != types.ImageStatePending {
      if imageState == types.ImageStateAvailable {
        c.amiID = *createImageOutput.ImageId
        return nil
      }
      return fmt.Errorf("created image %s (%q) is in invalid state %s", *createImageOutput.ImageId, c.rc.GetAMIName(), imageState)
    }
    time.Sleep(c.rc.UpdateTick)
  }
  return fmt.Errorf("the image %s (%q) didn't become available in %v", *createImageOutput.ImageId, c.rc.GetAMIName(), c.rc.UpdateTimeout)
}
