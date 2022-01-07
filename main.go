package main

import (
  "context"
  "flag"
  "log"
  "main/aws"
  "strings"
  "time"
)

func initRunConfig() *aws.RunConfig {
  groupName := flag.String("group", "", "the name of the Auto Scaling group to create; required.")
  instanceID := flag.String("instance", "", "AWS EC2 instance ID to create the service from; required.")
  healthPath := flag.String("health-path", "/health", "the health HTTP handler for the service.")
  daemonPort := flag.Int("port", 80, "the HTTP traffic port for the service.")
  instancesCount := flag.Int("instances", 1, "the number of instances to create within the group; min instances count and desired instances count will be set up to this value, max instances count will be set up to twice this value.")
  healthCheckGracePeriodStr := flag.String("health-check-grace-period", "1m", "the time needed for the instance to become healthy after the launch. Use the Golang duration strings to override, see https://pkg.go.dev/time#ParseDuration.")
  updateTimeoutStr := flag.String("update-timeout", "30m", "the time limit to complete the instance refresh; optional, default: 30m. Use the Golang duration strings to override, see https://pkg.go.dev/time#ParseDuration.")
  updateTickStr := flag.String("update-tick", "1m", "the time between status updates in the log file; optional, default: 1m. Making this parameter lower might speed up the overall execution. Use the Golang duration strings to override, see https://pkg.go.dev/time#ParseDuration.")
  flag.Parse()

  updateTimeout, err := time.ParseDuration(*updateTimeoutStr)
  if err != nil {
    log.Fatalf("cannot parse the update timeout string: %v", err)
  }
  updateTick, err := time.ParseDuration(*updateTickStr)
  if err != nil {
    log.Fatalf("cannot parse the update tick string: %v", err)
  }
  healthCheckGracePeriod, err := time.ParseDuration(*healthCheckGracePeriodStr)
  if err != nil {
    log.Fatalf("cannot parse the health check grace period string: %v", err)
  }

  return &aws.RunConfig{
    InstanceID:             *instanceID,
    GroupName:              *groupName,
    HealthPath:             *healthPath,
    DaemonPort:             int32(*daemonPort),
    InstancesCount:         int32(*instancesCount),
    HealthCheckGracePeriod: healthCheckGracePeriod,
    UpdateTimeout:          updateTimeout,
    UpdateTick:             updateTick,
  }
}

// TODO: cleanup on failure
func main() {
  rc := initRunConfig()
  if err := rc.ValidateArtifactNames(); err != nil {
    log.Fatalln(err)
  }
  client, err := aws.NewClient(context.Background(), rc)
  if err != nil {
    log.Fatalln(err)
  }
  instanceData, err := client.DescribeInstance()
  if err != nil {
    log.Fatalln(err)
  }
  defaultVPCID, err := client.GetDefaultVPCID()
  if err != nil {
    log.Fatalln(err)
  }
  subnetIDs, err := client.GetSubnets(defaultVPCID)
  if err != nil {
    log.Fatalln(err)
  }
  log.Printf("will create an AMI %q from the instance %s", rc.GetAMIName(), rc.InstanceID)
  log.Printf("will create a launch template %q", rc.GetLaunchTemplateName())
  log.Printf("will create a target group %q in VPC %s", rc.GetTargetGroupName(), defaultVPCID)
  log.Printf("will create a load balancer %q in subnets %s", rc.GetBalancerName(), strings.Join(subnetIDs, ", "))
  log.Printf("will create an auto scaling group %q with %d %s spot instances", rc.GetGroupName(), rc.InstancesCount, instanceData.InstanceType)
  if err := client.CreateAMI(); err != nil {
    log.Fatalln(err)
  }
  if err := client.CreateLaunchTemplate(instanceData); err != nil {
    log.Fatalln(err)
  }
  if err := client.CreateTargetGroup(instanceData); err != nil {
    log.Fatalln(err)
  }
  if err := client.CreateLoadBalancer(subnetIDs); err != nil {
    log.Fatalln(err)
  }
  if err := client.CreateAutoScalingGroup(subnetIDs); err != nil {
    log.Fatalln(err)
  }
  client.ReportCreatedArtifacts()
}
