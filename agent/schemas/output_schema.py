"""Agent output schemas (strict)."""
from __future__ import annotations

from typing import Literal

Severity = Literal["LOW", "MODERATE", "HIGH", "VERY_HIGH", "EXTREME"]
Priority = Literal["LOW", "MEDIUM", "HIGH"]
Recheck = Literal[15, 30, 60, 120, 360]

OUTPUT_SCHEMA = {
    "type": "object",
    "required": [
        "severity",
        "priority_wards",
        "reasoning_summary",
        "key_factors",
        "recommended_actions",
        "public_advisory",
        "recheck_interval_minutes",
        "agent_run_id",
        "fallback_used",
    ],
    "properties": {
        "severity": {"type": "string", "enum": ["LOW", "MODERATE", "HIGH", "VERY_HIGH", "EXTREME"]},
        "priority_wards": {"type": "array", "items": {"type": "string"}},
        "reasoning_summary": {"type": "string"},
        "key_factors": {"type": "array", "items": {"type": "string"}},
        "recommended_actions": {
            "type": "array",
            "items": {
                "type": "object",
                "required": ["action", "priority", "target"],
                "properties": {
                    "action": {"type": "string"},
                    "priority": {"type": "string", "enum": ["LOW", "MEDIUM", "HIGH"]},
                    "target": {"type": "string"},
                },
            },
        },
        "public_advisory": {"type": "string"},
        "recheck_interval_minutes": {"type": "integer", "enum": [15, 30, 60, 120, 360]},
        "agent_run_id": {"type": "string"},
        "fallback_used": {"type": "boolean"},
    },
}