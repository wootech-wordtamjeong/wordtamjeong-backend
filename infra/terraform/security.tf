resource "aws_security_group" "app" {
  name_prefix = "${local.name}-"
  description = "Wordtamjeong application server"
  vpc_id      = aws_vpc.this.id

  ingress {
    description = "SSH from administrator address"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.admin_cidr]
  }

  ingress {
    description = "Wordtamjeong HTTP API"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = var.api_allowed_cidrs
  }

  egress {
    description = "Allow outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name = "${local.name}-sg"
  }
}
