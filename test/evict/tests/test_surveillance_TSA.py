import sys
from datetime import datetime, timedelta, UTC
import time
import logging

from evict_helper import EvictHelper
from query_helper import QueryHelper


def test_surveillance_TSA(qh: QueryHelper, eh: EvictHelper):
    logger = logging.getLogger("test_surveillance_TSA")

    logger.info("📋 Surveillance TSA test")

    t = datetime.now(UTC) + timedelta(seconds=1)

    logger.debug("Creating test TSA")
    sub = qh.create_surveillance_TSA(t)

    if not sub:
        logger.error("❌ Unable to create TSA")
        sys.exit(1)

    TSA_id: str = str(sub["surveilled_area"]["id"])

    logger.debug("Check that TSA exists")
    if not qh.get_surveillance_TSA(TSA_id):
        logger.error("❌ Unable to retrieve TSA after creation")
        sys.exit(1)

    logger.debug("Evicting subscriptions older than 1s")
    eh.evict_surveillance_TSAs("1s", delete=True)

    logger.debug("Check that TSA still exists")
    if not qh.get_surveillance_TSA(TSA_id):
        logger.error("❌ Test TSA shall still be present since not expired")
        sys.exit(1)

    logger.debug("Waiting 3s so the TSA expires")
    _ = sys.stdout.flush()
    time.sleep(3)

    logger.debug("Evicting subscriptions older than 1s in dry mode")
    eh.evict_surveillance_TSAs("1s", delete=False)

    logger.debug("Check that TSA still exists")
    if not qh.get_surveillance_TSA(TSA_id):
        logger.error("❌ Test TSA shall still be present since delete was set to false")
        sys.exit(1)

    logger.debug("Evicting subscriptions older than 1s")
    eh.evict_surveillance_subscriptions("1s", delete=True)

    logger.debug("Check that TSA still exists")
    if not qh.get_surveillance_TSA(TSA_id):
        logger.error(
            "❌ Test TSA shall still be present since we evicted subscriptions"
        )
        sys.exit(1)

    logger.debug("Evicting subscriptions older than 1s on another locality")
    eh.evict_surveillance_TSAs("1s", delete=True, locality="somethingelse")
    if not qh.get_surveillance_TSA(TSA_id):
        logger.error(
            "❌ Test TSA shall still be present since we used another locality"
        )
        sys.exit(1)

    logger.debug("Evicting subscriptions older than 1s")
    eh.evict_surveillance_TSAs("1s", delete=True)

    logger.debug("Check that TSA has been deleted")
    if qh.get_surveillance_TSA(TSA_id):
        logger.error("❌ Test subscription shall has been deleted by evict")
        sys.exit(1)

    logger.info("✅ Surveillance TSA test successful :)")
