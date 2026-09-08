"""Decision agent entry point.

Provides:
- reason(): orchestrates tool use + LLM call
- fallback_plan(): rule-based plan when LLM fails
"""
from __future__ import annotations

import uuid
from typing import Any

from agent.schemas.output_schema import OUTPUT_SCHEMA
from agent.prompts.system_prompt import SYSTEM_PROMPT
from agent.tools.tools import (
    ToolContext,
    get_ward_risk,
    get_forecast,
    get_thermal_stress,
    get_vulnerability,
    get_resources,
    get_previous_alerts,
)

AGENT_VERSION = "agent-v1.0.0"


def fallback_plan(ward_ids: list[str], risks: dict[str, dict]) -> dict[str, Any]:
    """Rule-based action plan used when LLM fails."""
    severity = "LOW"
    actions: list[dict[str, Any]] = []
    for wid in ward_ids:
        r = risks.get(wid, {"band": "LOW"})
        band = r.get("band", "LOW")
        if band == "EXTREME":
            severity = "EXTREME"
            actions.extend([
                {"action": "Activate cooling center", "priority": "HIGH", "target": wid},
                {"action": "Issue outdoor-work advisory", "priority": "HIGH", "target": wid},
                {"action": "Pre-position ORS and water", "priority": "MEDIUM", "target": wid},
            ])
        elif band == "VERY_HIGH":
            severity = max(severity, "VERY_HIGH", key=["LOW", "MODERATE", "HIGH", "VERY_HIGH", "EXTREME"].index)
            actions.append({"action": "Pre-position cooling materials", "priority": "MEDIUM", "target": wid})
        elif band == "HIGH":
            severity = max(severity, "HIGH", key=["LOW", "MODERATE", "HIGH", "VERY_HIGH", "EXTREME"].index)
            actions.append({"action": "Issue public heat advisory", "priority": "LOW", "target": wid})

    return {
        "severity": severity,
        "priority_wards": ward_ids,
        "reasoning_summary": "Rule-based fallback plan based on per-ward risk bands.",
        "key_factors": [],
        "recommended_actions": actions,
        "public_advisory": "Stay hydrated, avoid midday sun, check on elderly neighbors.",
        "recheck_interval_minutes": 60 if severity in ("HIGH", "VERY_HIGH", "EXTREME") else 360,
        "agent_run_id": str(uuid.uuid4()),
        "fallback_used": True,
    }


def reason(ctx: ToolContext, ward_ids: list[str]) -> dict[str, Any]:
    """Top-level agent reasoning entry.

    For MVP we return the deterministic fallback plan after collecting risks.
    Production wiring: call LLM with SYSTEM_PROMPT + tool definitions; if
    timeout/error, fall back to fallback_plan.
    """
    risks = {wid: get_ward_risk(ctx, wid) for wid in ward_ids}
    return fallback_plan(ward_ids, risks)