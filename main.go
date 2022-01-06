package main

import (
  "context"
  "flag"
  "log"
  "main/aws"
  "time"
)

func initRunConfig() *aws.RunConfig {
  groupName := flag.String("group", "", "the name of the Auto Scaling group to update; required")
  instanceID := flag.String("instance", "", "AWS EC2 instance ID to create the AMI from; optional: do not use if you already have an AMI ID. The created launch template will inherit most of the options from this instance, including kernel id, network interfaces, placement, license specifications, and more.")
  healthPath := flag.String("health-path", "/health", "the daemon's health check path for the ELB balancers")
  daemonPort := flag.Int("port", 80, "the daemon's traffic port for the ELB balancers")
  instancesCount := flag.Int("instances", 1, "the number of instances to create within the group; min instances count and desired instances count will be set up to this value, max instances count will be set up to twice this value.")
  healthCheckGracePeriodStr := flag.String("health-check-grace-period", "1m", "the time needed for the instance to become healthy after the launch.")
  updateTimeoutStr := flag.String("update-timeout", "30m", "the time limit to complete the instance refresh; optional: the default is 30 minutes. Use the Golang duration strings to override, see https://pkg.go.dev/time#ParseDuration.")
  updateTickStr := flag.String("update-tick", "1m", "the time between status updates in the log file; optional: the default is one minute. Making this parameter lower might speed up the overall execution. Use the Golang duration strings to override, see https://pkg.go.dev/time#ParseDuration.")
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
  client, err := aws.NewClient(context.Background(), rc)
  if err != nil {
    log.Fatalln(err)
  }
  instanceData, err := client.DescribeInstance()
  if err != nil {
    log.Fatalln(err)
  }
  amiID, err := client.CreateAMI()
  if err != nil {
    log.Fatalln(err)
  }
  launchTemplateID, err := client.CreateLaunchTemplate(amiID, instanceData)
  if err != nil {
    log.Fatalln(err)
  }
  if err := client.CreateAutoScalingGroup(launchTemplateID, instanceData); err != nil {
    log.Fatalln(err)
  }
}
