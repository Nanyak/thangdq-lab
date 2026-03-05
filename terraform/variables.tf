variable "aws_region" {
  description = "The AWS region to deploy resources in"
  type = string
  default = "us-east-1"
}

variable "project_name" {
  description = "Name of the project for tagging"
  type = string
}
