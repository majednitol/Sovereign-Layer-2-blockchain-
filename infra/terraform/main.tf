terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# 1. VPC Configuration
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${var.project_name}-vpc"
  }
}

# Private Subnet for DB/Replication
resource "aws_subnet" "private" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "${var.aws_region}a"

  tags = {
    Name = "${var.project_name}-private-subnet"
  }
}

# Public Subnet for Gateway/Envoy
resource "aws_subnet" "public" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = "${var.aws_region}a"

  tags = {
    Name = "${var.project_name}-public-subnet"
  }
}

# 2. EKS Cluster (workloads)
resource "aws_eks_cluster" "eks" {
  name     = "${var.project_name}-cluster"
  role_arn = var.eks_role_arn

  vpc_config {
    subnet_ids = [aws_subnet.private.id, aws_subnet.public.id]
  }
}

# 3. RDS PostgreSQL Instance
resource "aws_db_instance" "rds" {
  allocated_storage   = 200
  db_name             = "sovereign_write"
  engine              = "postgres"
  engine_version      = "16"
  instance_class      = "db.r6g.xlarge"
  username            = "postgres"
  password            = "sovereign_admin_pwd"
  skip_final_snapshot = true
}

# 4. S3 Bucket for PITR Backups
resource "aws_s3_bucket" "backups" {
  bucket        = "${var.project_name}-backups-${var.environment}"
  force_destroy = true

  tags = {
    Name        = "${var.project_name}-backups"
    Environment = var.environment
  }
}
