#!/usr/bin/env python3
"""
Environment Manager for InvestorCenter

Smart environment configuration management without bash scripts.
Automatically detects and configures database connections for local vs production.
"""

import os
import sys
import json
import subprocess
import base64
from pathlib import Path
from typing import Dict, Optional
from dataclasses import dataclass


@dataclass
class DatabaseConfig:
    """Database configuration for different environments."""
    host: str
    port: int
    user: str
    password: str
    database: str
    sslmode: str

    @property
    def connection_string(self) -> str:
        """Get PostgreSQL connection string."""
        return f"postgresql://{self.user}:{self.password}@{self.host}:{self.port}/{self.database}?sslmode={self.sslmode}"

    def as_env_dict(self) -> Dict[str, str]:
        """Convert to environment variables dictionary."""
        return {
            'DB_HOST': self.host,
            'DB_PORT': str(self.port),
            'DB_USER': self.user,
            'DB_PASSWORD': self.password,
            'DB_NAME': self.database,
            'DB_SSLMODE': self.sslmode,
        }


class EnvironmentManager:
    """Manages environment configurations for InvestorCenter."""

    def __init__(self):
        self.project_root = Path(__file__).parent.parent
        self.configs = {
            'local': self._get_local_config(),
            'prod': self._get_prod_config(),
        }

    def _get_local_config(self) -> DatabaseConfig:
        """Get local development database configuration."""
        return DatabaseConfig(
            host='localhost',
            port=5432,
            user='investorcenter',
            password='investorcenter123',
            database='investorcenter_db',
            sslmode='disable'
        )

    def _get_prod_config(self) -> Optional[DatabaseConfig]:
        """Get production database configuration from Kubernetes."""
        try:
            # Check if kubectl is available
            subprocess.run(['kubectl', 'version', '--client'],
                          capture_output=True, check=True)

            # Check if namespace exists
            result = subprocess.run(
                ['kubectl', 'get', 'namespace', 'investorcenter'],
                capture_output=True
            )
            if result.returncode != 0:
                print("Production namespace 'investorcenter' not found")
                return None

            # Get password from Kubernetes secret
            result = subprocess.run([
                'kubectl', 'get', 'secret', 'postgres-secret',
                '-n', 'investorcenter',
                '-o', 'jsonpath={.data.password}'
            ], capture_output=True, text=True)

            if result.returncode != 0:
                print("Could not retrieve production database password")
                return None

            password = base64.b64decode(result.stdout).decode('utf-8')

            return DatabaseConfig(
                host='localhost',  # Will use port-forward
                port=5433,         # Port-forwarded port
                user='investorcenter',
                password=password,
                database='investorcenter_db',
                sslmode='disable'  # For port-forward access
            )

        except subprocess.CalledProcessError:
            print("kubectl not available or Kubernetes cluster not accessible")
            return None
        except Exception as e:
            print(f"Error getting production config: {e}")
            return None

    def start_port_forward(self) -> Optional[subprocess.Popen]:
        """Start port-forward for production database access."""
        try:
            # Kill any existing port-forwards
            subprocess.run(['pkill', '-f', 'kubectl port-forward.*postgres-service'],
                          capture_output=True)

            # Start new port-forward
            process = subprocess.Popen([
                'kubectl', 'port-forward', '-n', 'investorcenter',
                'svc/postgres-service', '5433:5432'
            ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

            # Give it time to start
            import time
            time.sleep(3)

            return process

        except Exception as e:
            print(f"Failed to start port-forward: {e}")
            return None

    def set_environment(self, env: str) -> bool:
        """Set environment variables for the specified environment."""
        if env not in self.configs:
            print(f"Unknown environment: {env}")
            print("Available environments: local, prod")
            return False

        config = self.configs[env]
        if config is None:
            print(f"Environment '{env}' is not available")
            return False

        # Set environment variables
        env_vars = config.as_env_dict()
        for key, value in env_vars.items():
            os.environ[key] = value

        # For production, start port-forward
        if env == 'prod':
            print("Starting port-forward to production database...")
            port_forward = self.start_port_forward()
            if port_forward is None:
                print("Failed to establish port-forward")
                return False

        print(f"Environment set to: {env}")
        print(f"Database: {config.host}:{config.port}")
        return True

    def get_current_environment(self) -> Optional[str]:
        """Detect current environment based on environment variables."""
        db_host = os.getenv('DB_HOST', '')
        db_port = os.getenv('DB_PORT', '')

        if db_host == 'localhost' and db_port == '5432':
            return 'local'
        elif db_host == 'localhost' and db_port == '5433':
            return 'prod'
        else:
            return None

    def test_connection(self, env: str) -> bool:
        """Test database connection for the specified environment."""
        config = self.configs.get(env)
        if config is None:
            return False

        try:
            # Set environment and test
            old_env = dict(os.environ)
            self.set_environment(env)

            # Import here to avoid loading issues
            sys.path.append(str(self.project_root / 'scripts'))
            from us_tickers.database import test_database_connection

            result = test_database_connection()

            # Restore environment
            os.environ.clear()
            os.environ.update(old_env)

            return result

        except Exception as e:
            print(f"Connection test failed: {e}")
            return False

    def print_status(self) -> None:
        """Print status of all environments."""
        print("InvestorCenter Environment Status")
        print("=================================")
        print()

        for env_name in ['local', 'prod']:
            config = self.configs[env_name]
            if config is None:
                print(f"{env_name.upper()}: NOT AVAILABLE")
                continue

            print(f"{env_name.upper()}:")
            print(f"  Host: {config.host}:{config.port}")
            print(f"  Database: {config.database}")
            print(f"  User: {config.user}")

            # Test connection
            if self.test_connection(env_name):
                print(f"  Status: CONNECTED")
                try:
                    # Get stock count if possible
                    self.set_environment(env_name)
                    sys.path.append(str(self.project_root / 'scripts'))
                    from us_tickers.database import get_database_stats
                    stats = get_database_stats()
                    if stats:
                        print(f"  Stocks: {stats.get('total_stocks', 0)}")
                except:
                    pass
            else:
                print(f"  Status: DISCONNECTED")
            print()


def main():
    """Main CLI interface."""
    import argparse

    parser = argparse.ArgumentParser(description='InvestorCenter Environment Manager')
    parser.add_argument('command', choices=['set', 'test', 'status', 'current'],
                       help='Command to execute')
    parser.add_argument('environment', nargs='?', choices=['local', 'prod'],
                       help='Target environment (local or prod)')

    args = parser.parse_args()

    manager = EnvironmentManager()

    if args.command == 'set':
        if not args.environment:
            print("Environment required for 'set' command")
            sys.exit(1)

        if manager.set_environment(args.environment):
            # Export environment variables for shell
            config = manager.configs[args.environment]
            if config:
                for key, value in config.as_env_dict().items():
                    print(f"export {key}='{value}'")
        else:
            sys.exit(1)

    elif args.command == 'test':
        if args.environment:
            success = manager.test_connection(args.environment)
            sys.exit(0 if success else 1)
        else:
            # Test all environments
            all_good = True
            for env in ['local', 'prod']:
                result = manager.test_connection(env)
                print(f"{env}: {'PASS' if result else 'FAIL'}")
                if not result:
                    all_good = False
            sys.exit(0 if all_good else 1)

    elif args.command == 'status':
        manager.print_status()

    elif args.command == 'current':
        current = manager.get_current_environment()
        if current:
            print(current)
        else:
            print("No environment detected")
            sys.exit(1)


if __name__ == '__main__':
    main()
