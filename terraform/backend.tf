terraform {
  backend "s3" {
    bucket         = "stuffsy-terraform-bucket"
    key            = "state/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
  }
}
