### See More:

#### AWS OpenAPI Docs
```
https://aws.amazon.com/cn/documentation/
https://docs.aws.amazon.com/AWSEC2/latest/APIReference/
```

#### Refs
```
https://docs.aws.amazon.com/sdk-for-go/api/
```

#### EC2 
  - Regions:				https://docs.aws.amazon.com/general/latest/gr/rande.html#ec2_region
  - Instance-Types:			https://aws.amazon.com/cn/ec2/instance-types/
  - Pricing:				https://aws.amazon.com/cn/ec2/pricing/?p=ps

#### Limits
> 绝大部分限制配额都可以提交工单进行提高
```
每个可用区正在按需运行的EC2实例个数为20个
https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-resource-limits.html
https://ap-southeast-1.console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Limits:
```


#### Using Centos7 AMI:
```
Offical CentOS is being distributed as a marketplace AMI rather than a community AMI, so you can't use it through API,
it will cause an error like this:

In order to use this AWS Marketplace product you need to accept terms and subscribe.
blabla ...

so we use Amazon Linux 2 AMI instead (also yum based)
https://aws.amazon.com/cn/amazon-linux-2/release-notes/
```
