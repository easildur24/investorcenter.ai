# =============================================================================
# Notification Service Infrastructure
# SNS topic for price updates, SQS queue for the notification service,
# IAM roles for backend (SNS publish) and notification service (SQS consume),
# ECR repository, and DLQ.
# =============================================================================

# --- SNS Topic: Price Updates ---
# Backend publishes bulk stock price snapshots every ~5 seconds during market hours.
# SQS queue subscribes; notification service long-polls from SQS.

resource "aws_sns_topic" "price_updates" {
  name = "investorcenter-price-updates"

  tags = {
    Name        = "investorcenter-price-updates"
    Environment = var.environment
    Service     = "notification"
  }
}

# --- SQS Queue: Alert Evaluation ---
# Notification service long-polls this queue for price updates.

resource "aws_sqs_queue" "notification_alerts" {
  name                       = "investorcenter-notification-alerts"
  visibility_timeout_seconds = 30       # Must exceed processing time
  message_retention_seconds  = 345600   # 4 days
  receive_wait_time_seconds  = 20       # Long polling

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.notification_alerts_dlq.arn
    maxReceiveCount     = 3
  })

  tags = {
    Name        = "investorcenter-notification-alerts"
    Environment = var.environment
    Service     = "notification"
  }
}

# --- SQS Dead Letter Queue ---

resource "aws_sqs_queue" "notification_alerts_dlq" {
  name                      = "investorcenter-notification-alerts-dlq"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name        = "investorcenter-notification-alerts-dlq"
    Environment = var.environment
    Service     = "notification"
  }
}

# --- SNS → SQS Subscription ---

resource "aws_sns_topic_subscription" "notification_alerts_sqs" {
  topic_arn = aws_sns_topic.price_updates.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.notification_alerts.arn
}

# Allow SNS to write to SQS
resource "aws_sqs_queue_policy" "allow_sns" {
  queue_url = aws_sqs_queue.notification_alerts.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "sns.amazonaws.com" }
      Action    = "sqs:SendMessage"
      Resource  = aws_sqs_queue.notification_alerts.arn
      Condition = {
        ArnEquals = { "aws:SourceArn" = aws_sns_topic.price_updates.arn }
      }
    }]
  })
}

# --- IAM: Backend SNS Publish (IRSA) ---
# Allows the backend K8s pods to publish price updates to SNS.

resource "aws_iam_role" "backend_sns_publisher" {
  name = "investorcenter-backend-sns-publisher"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = aws_iam_openid_connect_provider.eks.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${replace(aws_iam_openid_connect_provider.eks.url, "https://", "")}:sub" = "system:serviceaccount:investorcenter:backend-sa"
        }
      }
    }]
  })

  tags = {
    Name        = "investorcenter-backend-sns-publisher"
    Environment = var.environment
    Service     = "notification"
  }
}

resource "aws_iam_role_policy" "backend_sns_publish" {
  name = "backend-sns-publish"
  role = aws_iam_role.backend_sns_publisher.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["sns:Publish"]
      Resource = [aws_sns_topic.price_updates.arn]
    }]
  })
}

# --- IAM: Notification Service SQS Consume (IRSA) ---
# Allows the notification service K8s pods to read from SQS.

resource "aws_iam_role" "notification_sqs_consumer" {
  name = "investorcenter-notification-sqs-consumer"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = aws_iam_openid_connect_provider.eks.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${replace(aws_iam_openid_connect_provider.eks.url, "https://", "")}:sub" = "system:serviceaccount:investorcenter:notification-service-sa"
        }
      }
    }]
  })

  tags = {
    Name        = "investorcenter-notification-sqs-consumer"
    Environment = var.environment
    Service     = "notification"
  }
}

resource "aws_iam_role_policy" "notification_sqs_consume" {
  name = "notification-sqs-consume"
  role = aws_iam_role.notification_sqs_consumer.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "sqs:ReceiveMessage",
        "sqs:DeleteMessage",
        "sqs:GetQueueAttributes",
        "sqs:ChangeMessageVisibility"
      ]
      Resource = [aws_sqs_queue.notification_alerts.arn]
    }]
  })
}

# --- ECR Repository ---

resource "aws_ecr_repository" "notification_service" {
  name                 = "investorcenter/notification-service"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Name        = "investorcenter-notification-service"
    Environment = var.environment
    Service     = "notification"
  }
}

# --- Outputs ---

output "sns_price_updates_arn" {
  description = "ARN of the price updates SNS topic (set as SNS_PRICE_UPDATES_ARN in backend)"
  value       = aws_sns_topic.price_updates.arn
}

output "sqs_notification_alerts_url" {
  description = "URL of the SQS queue (set as SQS_QUEUE_URL in notification service)"
  value       = aws_sqs_queue.notification_alerts.id
}

output "backend_sns_publisher_role_arn" {
  description = "ARN of the IAM role for backend SNS publishing (annotate backend-sa with this)"
  value       = aws_iam_role.backend_sns_publisher.arn
}

output "notification_sqs_consumer_role_arn" {
  description = "ARN of the IAM role for notification service SQS consuming (annotate notification-service-sa with this)"
  value       = aws_iam_role.notification_sqs_consumer.arn
}

output "ecr_notification_service_url" {
  description = "URL of the notification service ECR repository"
  value       = aws_ecr_repository.notification_service.repository_url
}
