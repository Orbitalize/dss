import json
import sys
from urllib import request, error
from datetime import datetime, UTC
import uuid
import logging
import random


class QueryHelper:
    def __init__(self):
        self.token = self.get_token()
        self.logger = logging.getLogger(__name__)

    def get_token(self):
        scopes = [
            "utm.strategic_coordination",
            "utm.constraint_management",
            "rid.display_provider",
            "rid.service_provider",
        ]

        r = request.urlopen(
            f"http://localhost:8085/token?grant_type=client_credentials&scope={'%20'.join(scopes)}&intended_audience=localhost&issuer=localhost&sub=test_evict"
        ).read()
        data = json.loads(r)

        if not "access_token":
            self.logger.error(
                "❌ Unable to retrieve access token. Is the dummy auth server running?"
            )
            sys.exit(1)

        return data["access_token"]

    def do_dss_put_query(self, url, data):
        req = request.Request(
            url,
            data=json.dumps(data).encode("utf-8"),
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self.token}",
            },
            method="PUT",
        )

        return request.urlopen(req)

    def do_dss_get_query(self, url):
        req = request.Request(
            url,
            headers={"Authorization": f"Bearer {self.token}"},
            method="GET",
        )

        return request.urlopen(req)

    def create_scd_subscription(self, when):
        r = self.do_dss_put_query(
            f"http://localhost:8082/dss/v1/subscriptions/{uuid.uuid4()}",
            {
                "notify_for_operational_intents": True,
                "notify_for_constraints": False,
                "uss_base_url": "https://example.com/foo",
                "extents": {
                    "volume": {
                        "altitude_upper": {
                            "units": "M",
                            "reference": "W84",
                            "value": 300,
                        },
                        "altitude_lower": {
                            "units": "M",
                            "reference": "W84",
                            "value": 0,
                        },
                        "outline_circle": {
                            "radius": {"units": "M", "value": 100},
                            "center": {"lat": 22.910168434185902, "lng": 56},
                        },
                    },
                    "time_end": {"value": when.isoformat(), "format": "RFC3339"},
                },
            },
        )

        if r.status != 200:
            return None

        return json.loads(r.read())

    def get_scd_subscription(self, id):
        try:
            r = self.do_dss_get_query(
                f"http://localhost:8082/dss/v1/subscriptions/{id}"
            )
        except error.HTTPError as e:
            if e.code == 404:
                return None
            raise

        if r.status != 200:
            return None

        return json.loads(r.read())

    def create_scd_op_intent(self, when):
        r = self.do_dss_put_query(
            f"http://localhost:8082/dss/v1/operational_intent_references/{uuid.uuid4()}",
            {
                "state": "Accepted",
                "uss_base_url": "https://localhost:8080",
                "extents": [
                    {
                        "volume": {
                            "altitude_upper": {
                                "units": "M",
                                "reference": "W84",
                                "value": 300,
                            },
                            "altitude_lower": {
                                "units": "M",
                                "reference": "W84",
                                "value": 0,
                            },
                            "outline_circle": {
                                "radius": {"units": "M", "value": 100},
                                "center": {"lat": 22.910168434185902, "lng": 56},
                            },
                        },
                        "time_start": {
                            "value": datetime.now(UTC).isoformat(),
                            "format": "RFC3339",
                        },
                        "time_end": {"value": when.isoformat(), "format": "RFC3339"},
                    }
                ],
                "key": [],
            },
        )

        if r.status != 201:
            return None

        return json.loads(r.read())

    def get_scd_op_intent(self, id):
        try:
            r = self.do_dss_get_query(
                f"http://localhost:8082/dss/v1/operational_intent_references/{id}"
            )
        except error.HTTPError as e:
            if e.code == 404:
                return None
            raise

        if r.status != 200:
            return None

        return json.loads(r.read())

    def create_rid_subscription(self, when):
        r = self.do_dss_put_query(
            f"http://localhost:8082/rid/v2/dss/subscriptions/{uuid.uuid4()}",
            {
                "uss_base_url": "https://example.com/foo",
                "extents": {
                    "volume": {
                        "altitude_upper": {
                            "units": "M",
                            "reference": "W84",
                            "value": 300,
                        },
                        "altitude_lower": {
                            "units": "M",
                            "reference": "W84",
                            "value": 0,
                        },
                        "outline_circle": {
                            "radius": {"units": "M", "value": 100},
                            "center": {
                                "lat": random.uniform(-90, 90),
                                "lng": random.uniform(-180, 180),
                            },  # We use a random location to avoid too many subscription in the same location
                        },
                    },
                    "time_end": {"value": when.isoformat(), "format": "RFC3339"},
                },
            },
        )

        if r.status != 200:
            return None

        return json.loads(r.read())

    def get_rid_subscription(self, id):
        try:
            r = self.do_dss_get_query(
                f"http://localhost:8082/rid/v2/dss/subscriptions/{id}"
            )
        except error.HTTPError as e:
            if e.code == 404:
                return None
            raise

        if r.status != 200:
            return None

        return json.loads(r.read())

    def create_rid_ISA(self, when):
        r = self.do_dss_put_query(
            f"http://localhost:8082/rid/v2/dss/identification_service_areas/{uuid.uuid4()}",
            {
                "uss_base_url": "https://example.com/foo",
                "extents": {
                    "volume": {
                        "altitude_upper": {
                            "units": "M",
                            "reference": "W84",
                            "value": 300,
                        },
                        "altitude_lower": {
                            "units": "M",
                            "reference": "W84",
                            "value": 0,
                        },
                        "outline_circle": {
                            "radius": {"units": "M", "value": 100},
                            "center": {
                                "lat": random.uniform(-90, 90),
                                "lng": random.uniform(-180, 180),
                            },  # We use a random location to avoid too many subscription in the same location
                        },
                    },
                    "time_end": {"value": when.isoformat(), "format": "RFC3339"},
                },
            },
        )

        if r.status != 200:
            return None

        return json.loads(r.read())

    def get_rid_ISA(self, id):
        try:
            r = self.do_dss_get_query(
                f"http://localhost:8082/rid/v2/dss/identification_service_areas/{id}"
            )
        except error.HTTPError as e:
            if e.code == 404:
                return None
            raise

        if r.status != 200:
            return None

        return json.loads(r.read())
