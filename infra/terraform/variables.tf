variable "aws_region" {
  type        = string
  description = "AWS region for deployment"
  default     = "us-east-1"
}

variable "project_name" {
  type        = string
  description = "Project name prefix"
  default     = "sovereign"
}

variable "environment" {
  type        = string
  description = "Target deployment environment"
  default     = "mainnet"
}

variable "eks_role_arn" {
  type        = string
  description = "ARN for the EKS Cluster Service Role"
  default     = "arn:aws:iam::123456789012:role/eks-service-role"
}
