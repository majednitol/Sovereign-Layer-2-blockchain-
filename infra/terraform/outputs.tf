output "vpc_id" {
  value       = aws_vpc.main.id
  description = "The VPC ID"
}

output "eks_cluster_name" {
  value       = aws_eks_cluster.eks.name
  description = "EKS Cluster Name"
}

output "eks_endpoint" {
  value       = aws_eks_cluster.eks.endpoint
  description = "EKS Endpoint URI"
}

output "rds_host" {
  value       = aws_db_instance.rds.address
  description = "RDS TimescaleDB Endpoint host"
}

output "s3_backup_bucket" {
  value       = aws_s3_bucket.backups.id
  description = "S3 backup bucket for WAL/PITR"
}
