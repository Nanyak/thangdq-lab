resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true

  tags = {
    Name = "${var.project_name}-vpc"
  }
}

resource "aws_s3_bucket" "stuffsy" {
  bucket = "stuffsy"

  lifecycle {
    prevent_destroy = true
  }
}
