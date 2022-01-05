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

func (c *Client) CreateAMI() (string, error) {
  amiName := c.rc.GroupName + " v1"
  createImageOutput, err := c.ec2Client.CreateImage(c.ctx, &ec2.CreateImageInput{
    InstanceId: aws.String(c.rc.InstanceID),
    Name:       aws.String(amiName),
    NoReboot:   aws.Bool(false),
  })
  if err != nil {
    return "", fmt.Errorf("cannot create an AMI from instance %s: %v", c.rc.InstanceID, err)
  }
  finishTime := time.Now().Add(c.rc.UpdateTimeout)
  for time.Now().Before(finishTime) {
    time.Sleep(c.rc.UpdateTick)
    imageState, err := c.getImageState(*createImageOutput.ImageId)
    if err != nil {
      log.Printf("cannot get image state: %v", err)
      continue
    }
    log.Printf("creating image %s (%s): %s", amiName, *createImageOutput.ImageId, imageState)
    if imageState != types.ImageStatePending {
      if imageState == types.ImageStateAvailable {
        return *createImageOutput.ImageId, nil
      }
      return "", fmt.Errorf("created image %s (%s) is in invalid state %s", amiName, *createImageOutput.ImageId, imageState)
    }
  }
  return "", fmt.Errorf("the image %s (%s) didn't become available in %v", amiName, *createImageOutput.ImageId, c.rc.UpdateTimeout)
}
