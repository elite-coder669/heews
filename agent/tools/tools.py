"""Tool implementations. Each tool wraps a backend read or write."""
from __future__ import annotations

from typing import Any


class ToolContext:
    """Holds references to backend read APIs.

    In production, replace with HTTP client / RPC to Go service.
    """

    def __init__(self, backend: Any = None) -> None:
        self.backend = backend


def get_ward_risk(ctx: ToolContext, ward_id: str) -> dict[str, Any]:
    return ctx.backend.get_ward_risk(ward_id) if ctx.backend else {"ward_id": ward_id, "score": 0, "band": "LOW"}


def get_forecast(ctx: ToolContext, ward_id: str, hours: int = 72) -> list[dict[str, Any]]:
    return ctx.backend.get_forecast(ward_id, hours) if ctx.backend else []


def get_thermal_stress(ctx: ToolContext, ward_id: str) -> dict[str, Any]:
    return ctx.backend.get_thermal_stress(ward_id) if ctx.backend else {"wbgt_c": 0.0, "utci_c": 0.0}


def get_vulnerability(ctx: ToolContext, ward_id: str) -> dict[str, Any]:
    return ctx.backend.get_vulnerability(ward_id) if ctx.backend else {}


def get_resources(ctx: ToolContext, ward_id: str) -> dict[str, Any]:
    return ctx.backend.get_resources(ward_id) if ctx.backend else {}


def get_previous_alerts(ctx: ToolContext, ward_id: str, days: int = 7) -> list[dict[str, Any]]:
    return ctx.backend.get_previous_alerts(ward_id, days) if ctx.backend else []


def create_alert(ctx: ToolContext, payload: dict[str, Any]) -> dict[str, Any]:
    return ctx.backend.create_alert(payload) if ctx.backend else {"ok": True, "id": "alert_stub"}


def generate_action_plan(ctx: ToolContext, payload: dict[str, Any]) -> dict[str, Any]:
    return ctx.backend.generate_action_plan(payload) if ctx.backend else {"ok": True, "id": "plan_stub"}