package main

import (
  "context"
  "flag"
  "log"
  "main/aws"
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
