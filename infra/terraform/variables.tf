variable "aws_region" {
  description = "AWS region in which to create the resources."
  type        = string
  default     = "ap-northeast-2"
}

variable "environment" {
  description = "Deployment environment name."
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "prod"], var.environment)
    error_message = "environment must be either dev or prod."
  }
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.20.0.0/16"
}

variable "public_subnet_cidr" {
  description = "CIDR block for the public subnet."
  type        = string
  default     = "10.20.1.0/24"
}

variable "admin_cidr" {
  description = "Administrator public IP in CIDR notation, for example 203.0.113.10/32."
  type        = string

  validation {
    condition     = can(cidrnetmask(var.admin_cidr)) && var.admin_cidr != "0.0.0.0/0"
    error_message = "admin_cidr must be a valid CIDR and must not expose SSH to the entire internet."
  }
}

variable "api_allowed_cidrs" {
  description = "CIDR blocks allowed to access the public API on port 8080. Narrow this in production."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "instance_type" {
  description = "EC2 instance type."
  type        = string
  default     = "t3.micro"
}

variable "ssh_key_name" {
  description = "Name of an existing EC2 key pair."
  type        = string
}

variable "root_volume_size" {
  description = "Root EBS volume size in GiB."
  type        = number
  default     = 16
}

variable "bedrock_model_arns" {
  description = "Bedrock model or inference-profile ARNs the EC2 application may invoke."
  type        = list(string)
}

variable "ecr_repository_name" {
  description = "ECR repository name."
  type        = string
  default     = "wordtamjeong"
}

variable "github_repository" {
  description = "GitHub repository allowed to assume the deployment role."
  type        = string
  default     = "kimjuho1559/wordtamjeong-backend"
}

variable "github_oidc_provider_arn" {
  description = "ARN of the account's existing token.actions.githubusercontent.com OIDC provider. Leave null to skip the GitHub deployment role."
  type        = string
  default     = null
  nullable    = true
}
