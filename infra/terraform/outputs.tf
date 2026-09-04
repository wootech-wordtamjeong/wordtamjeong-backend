output "server_public_ip" {
  description = "Elastic IP address of the application server."
  value       = aws_eip.app.public_ip
}

output "server_public_dns" {
  description = "Public DNS name of the application server."
  value       = aws_instance.app.public_dns
}

output "ecr_repository_name" {
  description = "ECR repository name used by GitHub Actions."
  value       = aws_ecr_repository.app.name
}

output "ecr_repository_url" {
  description = "Full ECR repository URL."
  value       = aws_ecr_repository.app.repository_url
}

output "ec2_instance_id" {
  description = "EC2 instance ID, useful for SSM Session Manager."
  value       = aws_instance.app.id
}

output "security_group_id" {
  description = "Application security group ID used for temporary GitHub runner SSH access."
  value       = aws_security_group.app.id
}

output "github_deploy_role_arn" {
  description = "IAM role ARN to store as the GitHub Actions AWS_ROLE_ARN secret, or null when disabled."
  value       = try(aws_iam_role.github_deploy[0].arn, null)
}
