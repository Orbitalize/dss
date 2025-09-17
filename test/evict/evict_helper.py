import sys
import subprocess
import logging
import os


class EvictHelper:
    def __init__(self):
        self.logger = logging.getLogger(__name__)

    def run_evict(
        self,
        scd_oir=False,
        scd_sub=False,
        rid_isa=False,
        rid_sub=False,
        scd_ttl=None,
        rid_ttl=None,
        locality="local_dev",
        delete=False,
    ):
        db_hostname = os.environ.get("DB_HOSTNAME", "local-dss-crdb")
        db_port = os.environ.get("DB_PORT", "26257")
        db_username = os.environ.get("DB_USERNAME", "root")

        command = [
            "docker",
            "exec",
            "dss_sandbox-local-dss-core-service-1",
            "db-manager",
            "evict",
            f"--scd_oir={str(scd_oir).lower()}",
            f"--scd_sub={str(scd_sub).lower()}",
            f"--rid_isa={str(rid_isa).lower()}",
            f"--rid_sub={str(rid_sub).lower()}",
            "--locality",
            locality,
            "--cockroach_host",
            db_hostname,
            "--cockroach_port",
            db_port,
            "--cockroach_user",
            db_username,
        ]

        if delete:
            command.append("--delete")

        if scd_ttl:
            command += [
                "--scd_ttl",
                str(scd_ttl).lower(),
            ]

        if rid_ttl:
            command += [
                "--rid_ttl",
                str(rid_ttl).lower(),
            ]

        process = subprocess.run(
            " ".join(command), shell=True, capture_output=True, timeout=5
        )

        if process.returncode != 0:
            self.logger.error("❌ Unable to run evict command")
            self.logger.error(process.stdout.decode("utf-8"))
            self.logger.error(process.stderr.decode("utf-8"))
            sys.exit(1)

    def evict_scd_operational_intents(self, delay, delete):
        self.run_evict(scd_oir=True, delete=delete, scd_ttl=delay)

    def evict_scd_subcriptions(self, delay, delete):
        self.run_evict(scd_sub=True, delete=delete, scd_ttl=delay)

    def evict_rid_ISAs(self, delay, delete, locality="local_dev"):
        self.run_evict(rid_isa=True, delete=delete, rid_ttl=delay, locality=locality)

    def evict_rid_subcriptions(self, delay, delete, locality="local_dev"):
        self.run_evict(rid_sub=True, delete=delete, rid_ttl=delay, locality=locality)
