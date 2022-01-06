# AWS Auto Scaling Groups Builder

[AWS Auto Scaling group](https://docs.aws.amazon.com/autoscaling/ec2/userguide/AutoScalingGroup.html) is a great way of
managing Amazon EC2 instances. AWS Auto Scaling group watches the corresponding instances' health and launches new
instances whenever needed: to replace instances that became unhealthy or to [scale the group](
https://docs.aws.amazon.com/autoscaling/ec2/userguide/scaling_plan.html). Furthermore, with an [AWS Elastic Load
Balancer](https://docs.aws.amazon.com/autoscaling/ec2/userguide/attach-load-balancer-asg.html) attached, AWS Auto
Scaling Group also takes ELB health checks into account, making it possible to replace the instances based on
service-level signals.

However, creating AWS Auto Scaling Groups might be tricky. To create a new group, one needs to perform many
actions using either [AWS Management Console](
https://docs.aws.amazon.com/awsconsolehelpdocs/latest/gsg/learn-whats-new.html) or [AWS CLI](
https://docs.aws.amazon.com/cli/latest/userguide/). In addition, the user needs to wait until the corresponding
resources become available for some steps. It might not be easy to recall all the steps required if preparing new
services is not a regular job.

So, this is a job for an automation tool, such as AWS Auto Scaling Groups Builder. It uses [AWS EC2 API](
https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Welcome.html), [AWS EC2 Auto Scaling API](
https://docs.aws.amazon.com/autoscaling/ec2/APIReference/Welcome.html), and [AWS ELB API](
https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/Welcome.html) to perform the following operations
automatically:
- register an AMI from a running instance;
- create a launch template using this AMI;
- register a target group;
- create a load balancer with a listener that forwards traffic to this target group;
- create an Auto Scaling group with both EC2 and ELB health checks;
- handle all the errors that might happen along the way.

The tool uses the default AWS credentials config. Run `aws configure` or set up the environment variables
`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_REGION` before running the tool.

Only AWS EC2 Auto Scaling is supported. The service is supposed to handle health checks via HTTP.

The tool offers some default settings for all the services, which are not available for changing via command-line
arguments. However, it's always possible to change them manually with the default AWS tooling after the service is
ready. These defaults are:
- All the instances of the created service inherit the following settings of the source instance:
    - instance type;
    - kernel ID;
    - key name;
    - licence specifications;
    - placement;
    - network interfaces.
- The service only uses spot instances, despite the source instance lifecycle option.
- The tool creates and AMI with name `$GROUP_NAME v1`. The launch template name is just `$GROUP_NAME`. The names of 
target group and balancer are both `$GROUP_NAME` with substitution: `-` instead of `_`. E.g., if you're going to create
an Auto Scaling group with name `my_service`, the following resources will be created:
    - AMI with `my_service v1`;
    - launch template `my_service`;
    - target group `my-service`;
    - load balancer `my-service`;
    - Auto Scaling group `my_service`.
- The balancer is created in all availability zones, in the default subnets. So, if one has six availability zones in
their AWS account, all these zones will be used for placing the load balancer.
- The instances for the service are also placed within the default subnets of all the availability zones of the AWS
account.
- The Auto Scaling group has the following capacity settings:
    - `maximum capacity = 2 * desired capacity = 2 * minimum capacity`;
    - [capacity rebalancing](
https://docs.aws.amazon.com/autoscaling/ec2/userguide/ec2-auto-scaling-capacity-rebalancing.html) is turned on;
    - no dynamic or predictive scaling policies, no scheduled actions. 

## Examples of Use

The program options to create a new AMI from the instance `i-0d8767eed40de1728`, name it `service_ami`,
update the Auto Scaling group `my_service_group` with it:

`aws_asg_builder --instance i-0699803d818227e16 --group my_service_group --port 8080 --instances 10`

The output looks like that then:

```
2022/01/06 21:07:18 ami-02017ea24a4c875fe ("my_service_group v1"): pending
2022/01/06 21:08:18 ami-02017ea24a4c875fe ("my_service_group v1"): pending
2022/01/06 21:09:18 ami-02017ea24a4c875fe ("my_service_group v1"): pending
2022/01/06 21:10:19 ami-02017ea24a4c875fe ("my_service_group v1"): available
2022/01/06 21:10:19 created launch template "my_service_group"
2022/01/06 21:10:24 created target group "my-service-group" (arn:aws:elasticloadbalancing:us-east-1:046898261394:targetgroup/my-service-group/2a26fc5d5b6b6218)
2022/01/06 21:10:24 load balancer subnets: subnet-7e353733, subnet-bb0d39b5, subnet-6bafd034, subnet-6b7c0e0d, subnet-66ca4b57, subnet-dd8ef7fc
2022/01/06 21:10:26 load balancer "my-service-group": provisioning
2022/01/06 21:11:26 load balancer "my-service-group": provisioning
2022/01/06 21:12:26 load balancer "my-service-group": provisioning
2022/01/06 21:13:26 load balancer "my-service-group": active
2022/01/06 21:13:30 group "my_service_group": 0 instances in total, 0 instances are in service and healthy (10 needed)
2022/01/06 21:14:31 group "my_service_group", instance "i-054bf6155609ec8bc": InService, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-056ad0b0c87b9d83d": InService, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-082efd3e596a48fc0": Pending, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-08f77391f5024636d": Pending, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-0965bc3ae36c7f581": InService, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-0a3c909db0d21ccfd": InService, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-0ac330d72e4093cb4": Pending, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-0dcf5d90436696e75": Pending, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-0df393c4f980f582a": Pending, Healthy
2022/01/06 21:14:31 group "my_service_group", instance "i-0fec2014c2fc1706a": InService, Healthy
2022/01/06 21:14:31 group "my_service_group": 10 instances in total, 5 instances are in service and healthy (10 needed)
2022/01/06 21:15:32 group "my_service_group", instance "i-054bf6155609ec8bc": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-056ad0b0c87b9d83d": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-082efd3e596a48fc0": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-08f77391f5024636d": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-0965bc3ae36c7f581": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-0a3c909db0d21ccfd": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-0ac330d72e4093cb4": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-0dcf5d90436696e75": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-0df393c4f980f582a": InService, Healthy
2022/01/06 21:15:32 group "my_service_group", instance "i-0fec2014c2fc1706a": InService, Healthy
2022/01/06 21:15:32 group "my_service_group": 10 instances in total, 10 instances are in service and healthy (10 needed)
2022/01/06 21:15:32 successfully created an auto scaling group "my_service_group"
2022/01/06 21:15:32 check out the health status: http://my-service-group-1253581427.us-east-1.elb.amazonaws.com:8080/health
2022/01/06 21:15:33 enabled metrics collection for the group "my_service_group"
```

## Program Arguments

- `group`: the name of the Auto Scaling group to create; required.
- `instance`: AWS EC2 instance ID to create the service from; required.
- `health-path`: the health HTTP handler for the service; optional, default: `/health`.
- `port`: the HTTP traffic port for the service; optional, default: `80`.
- `instances`: the number of instances to create within the group; optional, default: `1`.
- `health-check-grace-period`: the time needed for the instance to become healthy after the launch; optional, default: `1m`. Use the Golang duration strings to override, see https://pkg.go.dev/time#ParseDuration. 
- `update-timeout`: the time limit to complete the instance refresh; optional, default: `30m`. Use the Golang duration strings to override, see https://pkg.go.dev/time#ParseDuration.
- `update-tick`: the time between status updates in the log file; optional, default: `1m`. Making this parameter lower might speed up the overall execution. Use the Golang duration strings to override, see https://pkg.go.dev/time#ParseDuration.

## Installation

Assuming you already have Golang installed on the machine, simply run:

```
go install github.com/ashagraev/aws_asg_builder@latest
```

The tool will then appear in your golang binary folder (e.g., `~/go/bin/aws_asg_builder`). If you don't have Golang yet,
consider installing it using the official guide https://go.dev/doc/install.

Alternatively, you can download the pre-built binaries from the latest release:
https://github.com/ashagraev/aws_asg_builder/releases/latest.
