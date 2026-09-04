locals {
  name = "wordtamjeong-${var.environment}"

  common_tags = {
    Project     = "wordtamjeong"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
