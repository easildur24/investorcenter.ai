#!/usr/bin/env python3
"""
Kubernetes CronJob Monitor Service

Watches Kubernetes Job events and automatically logs executions to the database.
"""

import os
import sys
import time
import logging
from datetime import datetime
from typing import Optional, Dict, Any

import psycopg2
from psycopg2.extras import RealDictCursor
from kubernetes import client, config, watch

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class CronJobMonitor:
    """Monitors Kubernetes Jobs and logs execution to database."""

    def __init__(self):
        """Initialize the monitor."""
        self.namespace = os.getenv('NAMESPACE', 'investorcenter')
        self.db_conn = None
        self.batch_v1 = None
        self.core_v1 = None

        # Track jobs we're currently monitoring
        self.tracked_jobs: Dict[str, Dict[str, Any]] = {}

    def connect_db(self):
        """Connect to PostgreSQL database."""
        try:
            self.db_conn = psycopg2.connect(
                host=os.getenv('DB_HOST', 'postgres-simple-service'),
                port=int(os.getenv('DB_PORT', '5432')),
                user=os.getenv('DB_USER', 'postgres'),
                password=os.getenv('DB_PASSWORD'),
                database=os.getenv('DB_NAME', 'investorcenter_db'),
                sslmode=os.getenv('DB_SSLMODE', 'disable'),
                cursor_factory=RealDictCursor
            )
            logger.info("Connected to database")
        except Exception as e:
            logger.error(f"Failed to connect to database: {e}")
            raise

    def init_kubernetes(self):
        """Initialize Kubernetes client."""
        try:
            # Try in-cluster config first
            config.load_incluster_config()
            logger.info("Loaded in-cluster Kubernetes config")
        except config.ConfigException:
            # Fall back to kubeconfig for local development
            config.load_kube_config()
            logger.info("Loaded kubeconfig")

        self.batch_v1 = client.BatchV1Api()
        self.core_v1 = client.CoreV1Api()

    def get_job_category(self, job_name: str) -> str:
        """Determine job category from job name."""
        if job_name.startswith('ic-score-'):
            return 'ic_score_pipeline'
        elif job_name.startswith('sec-filing-') or job_name.startswith('polygon-') or job_name.startswith('reddit-'):
            return 'core_pipeline'
        return 'other'

    def extract_cronjob_name(self, job_name: str) -> Optional[str]:
        """Extract the cronjob name from a job name (removes timestamp suffix)."""
        # Kubernetes appends a timestamp like -28405797 to cronjob job names
        # We need to remove this to get the cronjob name
        parts = job_name.rsplit('-', 1)
        if len(parts) == 2 and parts[1].isdigit():
            return parts[0]
        return job_name

    def get_pod_logs_summary(self, job_name: str) -> Optional[str]:
        """Get a summary from pod logs (first 500 chars of last 10 lines)."""
        try:
            pods = self.core_v1.list_namespaced_pod(
                namespace=self.namespace,
                label_selector=f'job-name={job_name}'
            )

            if not pods.items:
                return None

            pod_name = pods.items[0].metadata.name
            logs = self.core_v1.read_namespaced_pod_log(
                name=pod_name,
                namespace=self.namespace,
                tail_lines=10
            )

            # Return last 500 chars
            return logs[-500:] if logs else None

        except Exception as e:
            logger.warning(f"Failed to get pod logs for {job_name}: {e}")
            return None

    def log_job_execution(self, job_name: str, status: str,
                         started_at: datetime, completed_at: Optional[datetime] = None,
                         error_message: Optional[str] = None, pod_name: Optional[str] = None,
                         exit_code: Optional[int] = None):
        """Log job execution to database."""
        cronjob_name = self.extract_cronjob_name(job_name)
        job_category = self.get_job_category(cronjob_name)

        # Calculate duration if completed
        duration_seconds = None
        if completed_at and started_at:
            duration_seconds = int((completed_at - started_at).total_seconds())

        try:
            with self.db_conn.cursor() as cur:
                # Check if this execution already exists
                cur.execute(
                    "SELECT id FROM cronjob_execution_logs WHERE execution_id = %s",
                    (job_name,)
                )
                existing = cur.fetchone()

                if existing:
                    # Update existing record
                    cur.execute("""
                        UPDATE cronjob_execution_logs
                        SET status = %s,
                            completed_at = %s,
                            duration_seconds = %s,
                            error_message = %s,
                            exit_code = %s
                        WHERE execution_id = %s
                    """, (status, completed_at, duration_seconds, error_message, exit_code, job_name))
                    logger.info(f"Updated execution log for {job_name}: status={status}")
                else:
                    # Insert new record
                    cur.execute("""
                        INSERT INTO cronjob_execution_logs
                        (job_name, job_category, execution_id, status, started_at,
                         completed_at, duration_seconds, error_message, k8s_pod_name,
                         k8s_namespace, exit_code)
                        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                    """, (cronjob_name, job_category, job_name, status, started_at,
                          completed_at, duration_seconds, error_message, pod_name,
                          self.namespace, exit_code))
                    logger.info(f"Created execution log for {job_name}: status={status}")

                self.db_conn.commit()

        except Exception as e:
            logger.error(f"Failed to log job execution for {job_name}: {e}")
            self.db_conn.rollback()

    def handle_job_event(self, event_type: str, job: Any):
        """Handle a job event."""
        job_name = job.metadata.name
        cronjob_name = self.extract_cronjob_name(job_name)

        # Only monitor jobs from registered cronjobs
        if not self.is_monitored_cronjob(cronjob_name):
            return

        status = self.get_job_status(job)
        started_at = job.status.start_time
        completed_at = job.status.completion_time

        # Get pod information
        pod_name = None
        exit_code = None
        error_message = None

        if job.status.failed and job.status.failed > 0:
            # Job failed - try to get error details
            error_logs = self.get_pod_logs_summary(job_name)
            if error_logs:
                error_message = error_logs

            # Try to get exit code from pod
            try:
                pods = self.core_v1.list_namespaced_pod(
                    namespace=self.namespace,
                    label_selector=f'job-name={job_name}'
                )
                if pods.items:
                    pod = pods.items[0]
                    pod_name = pod.metadata.name
                    if pod.status.container_statuses:
                        container_status = pod.status.container_statuses[0]
                        if container_status.state.terminated:
                            exit_code = container_status.state.terminated.exit_code
            except Exception as e:
                logger.warning(f"Failed to get pod details for {job_name}: {e}")

        # Log to database
        self.log_job_execution(
            job_name=job_name,
            status=status,
            started_at=started_at,
            completed_at=completed_at,
            error_message=error_message,
            pod_name=pod_name,
            exit_code=exit_code
        )

    def get_job_status(self, job: Any) -> str:
        """Determine job status from Kubernetes Job object."""
        if job.status.succeeded and job.status.succeeded > 0:
            return 'success'
        elif job.status.failed and job.status.failed > 0:
            return 'failed'
        elif job.status.active and job.status.active > 0:
            return 'running'
        else:
            return 'running'

    def is_monitored_cronjob(self, cronjob_name: str) -> bool:
        """Check if this cronjob should be monitored."""
        try:
            with self.db_conn.cursor() as cur:
                cur.execute(
                    "SELECT 1 FROM cronjob_schedules WHERE job_name = %s AND is_active = true",
                    (cronjob_name,)
                )
                return cur.fetchone() is not None
        except Exception as e:
            logger.error(f"Failed to check if {cronjob_name} is monitored: {e}")
            return False

    def backfill_existing_jobs(self):
        """Backfill logs for any existing jobs that aren't logged yet."""
        logger.info("Backfilling existing jobs...")
        try:
            jobs = self.batch_v1.list_namespaced_job(namespace=self.namespace)

            for job in jobs.items:
                # Only process jobs created by cronjobs
                if not job.metadata.owner_references:
                    continue

                is_cronjob = any(
                    ref.kind == 'CronJob'
                    for ref in job.metadata.owner_references
                )

                if not is_cronjob:
                    continue

                job_name = job.metadata.name
                cronjob_name = self.extract_cronjob_name(job_name)

                if not self.is_monitored_cronjob(cronjob_name):
                    continue

                # Check if already logged
                with self.db_conn.cursor() as cur:
                    cur.execute(
                        "SELECT 1 FROM cronjob_execution_logs WHERE execution_id = %s",
                        (job_name,)
                    )
                    if cur.fetchone():
                        continue  # Already logged

                # Log this job
                logger.info(f"Backfilling job: {job_name}")
                self.handle_job_event('ADDED', job)

        except Exception as e:
            logger.error(f"Failed to backfill existing jobs: {e}")

    def watch_jobs(self):
        """Watch for job events and log them."""
        logger.info(f"Starting to watch jobs in namespace: {self.namespace}")

        w = watch.Watch()

        while True:
            try:
                for event in w.stream(
                    self.batch_v1.list_namespaced_job,
                    namespace=self.namespace,
                    timeout_seconds=300  # Reconnect every 5 minutes
                ):
                    event_type = event['type']
                    job = event['object']

                    # Only process jobs created by cronjobs
                    if not job.metadata.owner_references:
                        continue

                    is_cronjob = any(
                        ref.kind == 'CronJob'
                        for ref in job.metadata.owner_references
                    )

                    if not is_cronjob:
                        continue

                    logger.debug(f"Event: {event_type} for job {job.metadata.name}")
                    self.handle_job_event(event_type, job)

            except Exception as e:
                logger.error(f"Error watching jobs: {e}")
                logger.info("Reconnecting in 10 seconds...")
                time.sleep(10)

    def run(self):
        """Run the monitor service."""
        logger.info("Starting CronJob Monitor Service")

        # Initialize connections
        self.connect_db()
        self.init_kubernetes()

        # Backfill existing jobs
        self.backfill_existing_jobs()

        # Start watching
        self.watch_jobs()


def main():
    """Main entry point."""
    monitor = CronJobMonitor()

    try:
        monitor.run()
    except KeyboardInterrupt:
        logger.info("Shutting down...")
        sys.exit(0)
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
