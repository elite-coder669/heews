"""HEEWS SRS PDF generator using ReportLab.

Produces a professionally designed, presentation-quality technical
architecture document for the heatwave early warning hackathon.

Run:
    python3 tools/generate_pdf.py

Output:
    docs/SRS.pdf
"""
from __future__ import annotations

import sys
from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.colors import HexColor
from reportlab.lib.enums import TA_CENTER, TA_LEFT, TA_JUSTIFY
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import mm
from reportlab.platypus import (
    BaseDocTemplate,
    Frame,
    PageBreak,
    PageTemplate,
    Paragraph,
    Spacer,
    Table,
    TableStyle,
    KeepTogether,
    KeepInFrame,
)
from reportlab.platypus.flowables import HRFlowable

# ──────────────────────────────────────────────────────────────────────────────
# Design system
# ──────────────────────────────────────────────────────────────────────────────

PRIMARY   = HexColor("#0F172A")  # deep navy
SECONDARY = HexColor("#2563EB")  # technical blue
ACCENT    = HexColor("#06B6D4")  # cyan
SUCCESS   = HexColor("#16A34A")  # green
WARNING   = HexColor("#F59E0B")  # amber
DANGER    = HexColor("#DC2626")  # red
EXTREME   = HexColor("#7C1D6F")  # deep purple
LIGHT_BG  = HexColor("#F8FAFC")
MID_BG    = HexColor("#E2E8F0")
TEXT   = HexColor("#1E293B")
MUTED  = HexColor("#64748B")
WHITE  = HexColor("#FFFFFF")
BAND_LOW      = SUCCESS
BAND_MODERATE = WARNING
BAND_HIGH     = HexColor("#EA580C")
BAND_VHIGH    = DANGER
BAND_EXTREME  = EXTREME

PAGE_W, PAGE_H = A4
MARGIN = 18 * mm

DOCS_DIR = Path(__file__).resolve().parent.parent / "docs"
DOCS_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT = DOCS_DIR / "SRS.pdf"


# ──────────────────────────────────────────────────────────────────────────────
# Styles
# ──────────────────────────────────────────────────────────────────────────────

def make_styles() -> dict:
    base = getSampleStyleSheet()
    s = {}
    s["cover_title"] = ParagraphStyle(
        "cover_title", parent=base["Normal"], fontName="Helvetica-Bold",
        fontSize=34, leading=40, alignment=TA_CENTER, textColor=PRIMARY,
        spaceAfter=4 * mm,
    )
    s["cover_sub"] = ParagraphStyle(
        "cover_sub", parent=base["Normal"], fontName="Helvetica",
        fontSize=14, leading=18, alignment=TA_CENTER, textColor=SECONDARY,
        spaceAfter=8 * mm,
    )
    s["cover_meta"] = ParagraphStyle(
        "cover_meta", parent=base["Normal"], fontName="Helvetica",
        fontSize=10, leading=14, alignment=TA_CENTER, textColor=MUTED,
    )
    s["chapter"] = ParagraphStyle(
        "chapter", parent=base["Heading1"], fontName="Helvetica-Bold",
        fontSize=22, leading=28, textColor=PRIMARY, spaceBefore=0, spaceAfter=8 * mm,
    )
    s["section"] = ParagraphStyle(
        "section", parent=base["Heading2"], fontName="Helvetica-Bold",
        fontSize=14, leading=18, textColor=SECONDARY, spaceBefore=6 * mm, spaceAfter=4 * mm,
    )
    s["subsection"] = ParagraphStyle(
        "subsection", parent=base["Heading3"], fontName="Helvetica-Bold",
        fontSize=12, leading=15, textColor=TEXT, spaceBefore=4 * mm, spaceAfter=2 * mm,
    )
    s["body"] = ParagraphStyle(
        "body", parent=base["Normal"], fontName="Helvetica", fontSize=10,
        leading=14, alignment=TA_JUSTIFY, textColor=TEXT, spaceAfter=3 * mm,
    )
    s["body_small"] = ParagraphStyle(
        "body_small", parent=base["Normal"], fontName="Helvetica", fontSize=9,
        leading=12, textColor=TEXT, spaceAfter=2 * mm,
    )
    s["muted"] = ParagraphStyle(
        "muted", parent=base["Normal"], fontName="Helvetica-Oblique",
        fontSize=9, leading=12, textColor=MUTED,
    )
    s["code"] = ParagraphStyle(
        "code", parent=base["Normal"], fontName="Courier", fontSize=8.5,
        leading=11, textColor=TEXT,
    )
    s["caption"] = ParagraphStyle(
        "caption", parent=base["Normal"], fontName="Helvetica-Oblique",
        fontSize=9, leading=12, textColor=MUTED, alignment=TA_CENTER,
        spaceAfter=4 * mm,
    )
    s["num_label"] = ParagraphStyle(
        "num_label", parent=base["Normal"], fontName="Helvetica-Bold",
        fontSize=11, leading=14, textColor=ACCENT, alignment=TA_LEFT,
    )
    s["toc_item"] = ParagraphStyle(
        "toc_item", parent=base["Normal"], fontName="Helvetica", fontSize=10,
        leading=15, textColor=TEXT,
    )
    return s


# ──────────────────────────────────────────────────────────────────────────────
# Document + page templates
# ──────────────────────────────────────────────────────────────────────────────

class HEEWSDoc(BaseDocTemplate):
    def __init__(self, filename, **kw):
        super().__init__(filename, pagesize=A4,
                         leftMargin=MARGIN, rightMargin=MARGIN,
                         topMargin=MARGIN, bottomMargin=MARGIN, **kw)
        frame = Frame(MARGIN, MARGIN, PAGE_W - 2 * MARGIN, PAGE_H - 2 * MARGIN,
                      id="normal", showBoundary=0)
        self.addPageTemplates([
            PageTemplate(id="cover", frames=[frame], onPage=_draw_cover_chrome),
            PageTemplate(id="normal", frames=[frame], onPage=_draw_chrome),
        ])
        self._page_kind = "cover"

    def afterFlowable(self, flowable):
        if isinstance(flowable, Paragraph):
            txt = flowable.getPlainText()
            if txt.startswith("CHAPTER::"):
                self.canv.bookmarkPage(txt.split("::", 1)[1])
                self.canv.addOutlineEntry(txt.split("::", 1)[1], txt.split("::", 1)[1], level=0)
            elif txt.startswith("SECTION::"):
                self.canv.addOutlineEntry(txt.split("::", 1)[1], txt.split("::", 1)[1], level=1)

    def set_page_kind(self, kind):
        self._page_kind = kind
        self.handle_pageBegin()  # refresh template choice


def _draw_chrome(canvas, doc):
    canvas.saveState()
    # Header
    canvas.setFont("Helvetica-Bold", 8)
    canvas.setFillColor(SECONDARY)
    canvas.drawString(MARGIN, PAGE_H - MARGIN + 10, "EXTREME HEATWAVE EARLY WARNING")
    canvas.setFont("Helvetica", 8)
    canvas.setFillColor(MUTED)
    canvas.drawRightString(PAGE_W - MARGIN, PAGE_H - MARGIN + 10,
                           "Technical Architecture & SRS")
    canvas.setStrokeColor(MID_BG)
    canvas.setLineWidth(0.4)
    canvas.line(MARGIN, PAGE_H - MARGIN + 6, PAGE_W - MARGIN, PAGE_H - MARGIN + 6)
    # Footer
    canvas.setFont("Helvetica", 8)
    canvas.setFillColor(MUTED)
    canvas.drawString(MARGIN, MARGIN - 12, "SIH Prototype  •  Version 1.0")
    canvas.drawRightString(PAGE_W - MARGIN, MARGIN - 12, f"Page {doc.page}")
    canvas.setStrokeColor(MID_BG)
    canvas.line(MARGIN, MARGIN - 6, PAGE_W - MARGIN, MARGIN - 6)
    canvas.restoreState()


def _draw_cover_chrome(canvas, doc):
    # Subtle band on cover
    canvas.saveState()
    canvas.setFillColor(PRIMARY)
    canvas.rect(0, PAGE_H - 18 * mm, PAGE_W, 18 * mm, stroke=0, fill=1)
    canvas.setFillColor(WHITE)
    canvas.setFont("Helvetica-Bold", 9)
    canvas.drawString(MARGIN, PAGE_H - 12 * mm, "HEEWS  •  HEATWAVE EARLY WARNING")
    canvas.drawRightString(PAGE_W - MARGIN, PAGE_H - 12 * mm, "TECHNICAL ARCHITECTURE  •  v1.0")
    canvas.restoreState()


# ──────────────────────────────────────────────────────────────────────────────
# Reusable components
# ──────────────────────────────────────────────────────────────────────────────

def cover_page(styles) -> list:
    elems: list = []
    elems.append(Spacer(1, 60 * mm))
    elems.append(Paragraph("EXTREME HEATWAVE<br/>EARLY WARNING", styles["cover_title"]))
    elems.append(Paragraph("&amp;", styles["cover_sub"]))
    elems.append(Paragraph("HUMAN THERMAL<br/>STRESS INDEX", styles["cover_title"]))
    elems.append(Spacer(1, 12 * mm))
    elems.append(Paragraph(
        "Technical Architecture  •  Software Requirements Specification",
        styles["cover_sub"]))
    elems.append(Paragraph(
        "ML + Physics + Agent + Backend + Frontend",
        styles["cover_sub"]))
    elems.append(Spacer(1, 30 * mm))
    elems.append(Paragraph("SIH Prototype  ·  Version 1.0  ·  September 2026", styles["cover_meta"]))
    elems.append(Paragraph("Decision-support prototype. Not medical advice.", styles["cover_meta"]))
    elems.append(PageBreak())
    return elems


def toc_page(styles) -> list:
    rows = [
        ("01", "Executive Summary", "3"),
        ("02", "Problem Understanding", "4"),
        ("03", "End-to-End System Explanation", "5"),
        ("04", "Goals and Non-Goals", "6"),
        ("05", "Software Requirements Specification", "7"),
        ("06", "Functional Requirements", "8"),
        ("07", "Non-Functional Requirements", "11"),
        ("08", "Architecture", "12"),
        ("09", "Domain Ownership", "14"),
        ("10", "Data Flow", "15"),
        ("11", "Weather System", "16"),
        ("12", "Spatial System", "17"),
        ("13", "Physics System", "18"),
        ("14", "ML / Risk System", "19"),
        ("15", "Agent System", "20"),
        ("16", "Backend System", "22"),
        ("17", "Frontend System", "23"),
        ("18", "Database", "24"),
        ("19", "API Contracts", "25"),
        ("20", "Agent Tool Contracts", "27"),
        ("21", "Monorepo Structure", "28"),
        ("22", "Scaffold Routes", "29"),
        ("23", "Environment Configuration", "30"),
        ("24", "Testing Strategy", "31"),
        ("25", "Failure Handling", "32"),
        ("26", "Observability", "33"),
        ("27", "Security & Safety", "34"),
        ("28", "24-Hour Execution Plan", "35"),
        ("29", "Team Parallelization Plan", "36"),
        ("30", "Domain-Specific Agent Prompts", "37"),
        ("31", "Demo Flow", "39"),
        ("32", "Competitive Differentiation", "40"),
        ("33", "Limitations", "41"),
        ("34", "Traceability Matrix", "42"),
        ("35", "Definition of Done", "43"),
    ]
    table_data = []
    for num, title, page in rows:
        table_data.append([
            Paragraph(f"<font color='#06B6D4'><b>{num}</b></font>", styles["toc_item"]),
            Paragraph(title, styles["toc_item"]),
            Paragraph(page, styles["toc_item"]),
        ])

    table = Table(table_data, colWidths=[18 * mm, 130 * mm, 18 * mm])
    table.setStyle(TableStyle([
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
        ("TOPPADDING", (0, 0), (-1, -1), 6),
        ("LINEBELOW", (0, 0), (-1, -1), 0.25, MID_BG),
    ]))

    elems = [
        Paragraph("TABLE OF CONTENTS", styles["chapter"]),
        HRFlowable(width="100%", thickness=0.6, color=MID_BG, spaceAfter=6 * mm),
        table,
        PageBreak(),
    ]
    return elems


def chapter_header(num: str, title: str, blurb: str, styles) -> list:
    elems = []
    elems.append(Paragraph(f"<font color='#06B6D4'>{num}</font>", styles["num_label"]))
    elems.append(Paragraph(title, styles["chapter"]))
    elems.append(Paragraph(f"<i>{blurb}</i>", styles["muted"]))
    elems.append(HRFlowable(width="100%", thickness=0.6, color=SECONDARY, spaceAfter=6 * mm))
    return elems


def info_box(title: str, body: str, styles, color: str = "#2563EB") -> list:
    bg = HexColor(color)
    data = [[
        Paragraph(f"<b><font color='white'>{title}</font></b>", styles["body_small"]),
        Paragraph(f"<font color='white'>{body}</font>", styles["body_small"]),
    ]]
    t = Table(data, colWidths=[28 * mm, 140 * mm])
    t.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, -1), bg),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("LEFTPADDING", (0, 0), (-1, -1), 8),
        ("RIGHTPADDING", (0, 0), (-1, -1), 8),
        ("TOPPADDING", (0, 0), (-1, -1), 8),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 8),
        ("BOX", (0, 0), (-1, -1), 0.3, bg),
    ]))
    return [t, Spacer(1, 3 * mm)]


def warning_box(title: str, body: str, styles) -> list:
    return info_box(title, body, styles, color="#F59E0B")


def danger_box(title: str, body: str, styles) -> list:
    return info_box(title, body, styles, color="#DC2626")


def success_box(title: str, body: str, styles) -> list:
    return info_box(title, body, styles, color="#16A34A")


def architecture_box(title: str, body: str, styles) -> list:
    return info_box(title, body, styles, color="#2563EB")


def decision_box(title: str, body: str, styles) -> list:
    return info_box(title, body, styles, color="#7C1D6F")


def kv_table(rows: list[tuple[str, str]], styles) -> Table:
    data = [[Paragraph(f"<b>{k}</b>", styles["body_small"]),
             Paragraph(v, styles["body_small"])] for k, v in rows]
    t = Table(data, colWidths=[55 * mm, 120 * mm])
    t.setStyle(TableStyle([
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("BACKGROUND", (0, 0), (0, -1), LIGHT_BG),
        ("LEFTPADDING", (0, 0), (-1, -1), 6),
        ("RIGHTPADDING", (0, 0), (-1, -1), 6),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
        ("LINEBELOW", (0, 0), (-1, -1), 0.3, MID_BG),
    ]))
    return t


def styled_table(header: list[str], rows: list[list[str]], styles, col_widths=None) -> Table:
    data = [[Paragraph(f"<b><font color='white'>{h}</font></b>", styles["body_small"]) for h in header]]
    for row in rows:
        data.append([Paragraph(c, styles["body_small"]) for c in row])
    t = Table(data, colWidths=col_widths)
    t.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, 0), PRIMARY),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("ROWBACKGROUNDS", (0, 1), (-1, -1), [WHITE, LIGHT_BG]),
        ("LEFTPADDING", (0, 0), (-1, -1), 6),
        ("RIGHTPADDING", (0, 0), (-1, -1), 6),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
        ("LINEBELOW", (0, 0), (-1, -1), 0.3, MID_BG),
    ]))
    return t


def code_block(text: str, styles) -> Table:
    p = Paragraph(text.replace(" ", "&nbsp;").replace("\n", "<br/>"), styles["code"])
    t = Table([[p]], colWidths=[170 * mm])
    t.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, -1), LIGHT_BG),
        ("LEFTPADDING", (0, 0), (-1, -1), 8),
        ("RIGHTPADDING", (0, 0), (-1, -1), 8),
        ("TOPPADDING", (0, 0), (-1, -1), 6),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
        ("BOX", (0, 0), (-1, -1), 0.3, MID_BG),
    ]))
    return t


def architecture_diagram(styles) -> Table:
    """Vertical stacked-component architecture diagram."""
    boxes = [
        ("Open-Meteo (LIVE) / fixtures (MOCK)", "#2563EB"),
        ("Weather Ingestion", "#2563EB"),
        ("Spatial Downscaling (IDW + centroids)", "#2563EB"),
        ("Ward Weather Table", "#0F172A"),
        ("Physics Engine (WBGT, UTCI)", "#0EA5E9"),
        ("Vulnerability / Demographics", "#8B5CF6"),
        ("Risk Model (MRI)", "#06B6D4"),
        ("Decision Agent (LLM + fallback)", "#7C1D6F"),
        ("Backend / Alerts API (Go)", "#16A34A"),
        ("GIS Dashboard (React)", "#F59E0B"),
    ]
    rows = []
    for label, color in boxes:
        p = Paragraph(f"<b><font color='white'>{label}</font></b>", styles["body_small"])
        rows.append([p])
        if label != boxes[-1][0]:
            rows.append([Paragraph("<font color='#64748B'>↓</font>", styles["body_small"])])
    t = Table(rows, colWidths=[170 * mm])
    style = [
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("ALIGN", (0, 0), (-1, -1), "CENTER"),
        ("TOPPADDING", (0, 0), (-1, -1), 6),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
        ("LEFTPADDING", (0, 0), (-1, -1), 8),
        ("RIGHTPADDING", (0, 0), (-1, -1), 8),
    ]
    row_idx = 0
    for _label, color in boxes:
        style.append(("BACKGROUND", (0, row_idx), (0, row_idx), HexColor(color)))
        row_idx += 2
    t.setStyle(TableStyle(style))
    return t


def risk_badge(band: str, styles) -> Table:
    color = {
        "LOW": BAND_LOW, "MODERATE": BAND_MODERATE, "HIGH": BAND_HIGH,
        "VERY HIGH": BAND_VHIGH, "EXTREME": BAND_EXTREME,
    }[band]
    p = Paragraph(f"<b><font color='white'>{band}</font></b>", styles["body_small"])
    t = Table([[p]], colWidths=[28 * mm])
    t.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, -1), color),
        ("ALIGN", (0, 0), (-1, -1), "CENTER"),
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ]))
    return t


# ──────────────────────────────────────────────────────────────────────────────
# Build the story
# ──────────────────────────────────────────────────────────────────────────────

def build_story(styles: dict) -> list:
    S = []
    # Cover and TOC
    S.extend(cover_page(styles))
    S.extend(toc_page(styles))

    # Chapter 1 — Executive Summary
    S.extend(chapter_header("01", "Executive Summary", "Understanding the transformation from weather to municipal action.", styles))
    S.append(Paragraph(
        "The <b>Heatwave Early Warning &amp; Human Thermal Stress Index (HEEWS)</b> system is a decision-support "
        "prototype for the city of Hyderabad. It pulls a 3–5 day weather forecast, converts it into per-ward "
        "human thermal-stress metrics (WBGT and UTCI), combines those with ward-level vulnerability data, and "
        "produces a population-level heat health risk index (MRI) per ward. A reasoning agent then recommends "
        "concrete municipal interventions.", styles["body"]))
    S.append(Paragraph(
        "It is <b>not</b> a clinical mortality predictor. It is a defensible, reproducible, audit-friendly "
        "decision-support system that helps a municipal command center decide where, when, and how to act.",
        styles["body"]))
    S.append(Paragraph("<b>Core transformation</b>", styles["subsection"]))
    S.extend(architecture_box("Core Transformation",
                              "WEATHER → PHYSIOLOGICAL HEAT STRESS → POPULATION RISK → DECISION → ACTION",
                              styles))
    S.append(Paragraph(
        "Every component in the architecture implements exactly one arrow in this chain.",
        styles["body"]))

    # Chapter 2 — Problem Understanding
    S.extend(chapter_header("02", "Problem Understanding",
                            "Why raw temperature warnings are insufficient for heat-health decisions.", styles))
    S.append(Paragraph(
        "Cities like Hyderabad experience extreme heatwaves that cause preventable mortality and morbidity, "
        "especially among outdoor workers, the elderly, and residents of informal housing. Most existing "
        "systems provide either raw temperature warnings or generic citywide advisories that miss the most "
        "vulnerable wards.", styles["body"]))
    S.append(Paragraph(
        "Air temperature alone is insufficient because human heat stress depends on humidity, wind, "
        "radiation, and pressure. Two days with identical air temperature can produce wildly different "
        "physiological outcomes. Ward-level resolution matters because Urban Heat Island intensity and "
        "vulnerability composition vary sharply between wards.", styles["body"]))

    # Chapter 3 — End-to-End Walkthrough
    S.extend(chapter_header("03", "End-to-End System Explanation",
                            "Walkthrough from forecast to municipal action.", styles))
    S.append(Paragraph(
        "Illustrative values for Ward A (clearly labeled as examples):",
        styles["body"]))
    S.append(styled_table(
        ["Variable", "Value", "Unit"],
        [
            ["Temperature", "41.2", "°C"],
            ["Relative humidity", "58", "%"],
            ["Wind speed", "2.1", "m/s"],
            ["Shortwave radiation", "710", "W/m²"],
            ["WBGT (Liljegren)", "33.8", "°C"],
            ["UTCI", "44.6", "°C"],
            ["MRI", "82 / 100", "—"],
            ["Band", "EXTREME", "—"],
        ],
        styles,
        col_widths=[55 * mm, 55 * mm, 55 * mm],
    ))
    S.append(Spacer(1, 4 * mm))
    S.extend(decision_box("Agent Reasoning (illustrative)",
                          "High thermal stress, high outdoor-worker share, persistent multi-day heat. "
                          "Recommend activating cooling center, issuing outdoor-work advisory, "
                          "and scheduling welfare checks.", styles))

    # Chapter 4 — Goals and Non-Goals
    S.extend(chapter_header("04", "Goals and Non-Goals",
                            "What this MVP must do, and what it explicitly does not.", styles))
    S.append(styled_table(
        ["#", "Goal", "Priority"],
        [
            ["G1", "Fetch Open-Meteo forecast for Hyderabad", "MUST"],
            ["G2", "Downscale to ward-level weather", "MUST"],
            ["G3", "Compute WBGT and UTCI per ward per hour", "MUST"],
            ["G4", "Generate 3–5 day forecast heat-health risk per ward", "MUST"],
            ["G5", "Produce explainable vulnerability-aware MRI", "MUST"],
            ["G6", "Reasoning agent proposes municipal actions", "MUST"],
            ["G7", "Render a GIS dashboard of Hyderabad ward risk", "MUST"],
            ["G8", "Public-friendly summary view", "SHOULD"],
            ["G9", "Run end-to-end on synthetic data offline", "MUST"],
            ["G10", "End-to-end demo runs in under 5 minutes", "MUST"],
        ],
        styles,
        col_widths=[15 * mm, 130 * mm, 25 * mm],
    ))
    S.append(Spacer(1, 4 * mm))
    S.append(Paragraph("<b>Non-Goals (out of scope for MVP)</b>", styles["subsection"]))
    S.extend(warning_box("Out of Scope",
                         "Clinical mortality prediction · individual diagnosis · real-time telemetry · "
                         "SMS/push delivery · authenticated workflows · cross-city federation.",
                         styles))

    # Chapter 5 — SRS
    S.extend(chapter_header("05", "Software Requirements Specification",
                            "Authoritative specification; this chapter summarizes the full document.", styles))
    S.append(Paragraph(
        "This chapter is the SRS core. Detailed schemas are in chapter 19 (API Contracts) and chapter 20 "
        "(Agent Tool Contracts). Requirements are tagged MUST, SHOULD, COULD, CUT.",
        styles["body"]))

    # Chapter 6 — Functional Requirements
    S.extend(chapter_header("06", "Functional Requirements",
                            "Numbered FR-### requirements across all domains.", styles))
    for label, header, rows in [
        ("FR-W", "Weather Ingestion", [
            ["FR-W-001", "Fetch current weather for Hyderabad bounding box", "MUST"],
            ["FR-W-002", "Fetch 3–5 day hourly forecast", "MUST"],
            ["FR-W-003", "Normalize units (km/h → m/s, hPa → Pa)", "MUST"],
            ["FR-W-004", "Validate presence of required fields", "MUST"],
            ["FR-W-005", "Store raw_weather_staging rows with source timestamp", "MUST"],
            ["FR-W-006", "Cache last-good response per pipeline_run_id", "MUST"],
            ["FR-W-007", "Support deterministic mock fixtures", "MUST"],
            ["FR-W-008", "Surface fetch errors with explicit codes", "MUST"],
        ]),
        ("FR-S", "Spatial Processing", [
            ["FR-S-001", "Load ward polygon GeoJSON", "MUST"],
            ["FR-S-002", "Validate polygon geometry", "MUST"],
            ["FR-S-003", "Assign stable ward_id (UUID)", "MUST"],
            ["FR-S-004", "Compute centroid latitude/longitude", "MUST"],
            ["FR-S-005", "Run IDW interpolation from grid → centroid", "MUST"],
            ["FR-S-006", "Handle missing weather points gracefully", "MUST"],
            ["FR-S-007", "Reject duplicate ward_number without unique stable_id", "MUST"],
        ]),
        ("FR-P", "Physics Engine", [
            ["FR-P-001", "Compute solar geometry", "MUST"],
            ["FR-P-002", "Compute WBGT using Liljegren formulation", "MUST"],
            ["FR-P-003", "Compute simplified WBGT fallback", "MUST"],
            ["FR-P-004", "Compute UTCI", "MUST"],
            ["FR-P-005", "Validate all physics inputs", "MUST"],
            ["FR-P-006", "Emit thermal_stress JSON with explicit units", "MUST"],
            ["FR-P-007", "Pin physics_version string per run", "MUST"],
            ["FR-P-008", "Reference test cases for known WBGT/UTCI", "MUST"],
        ]),
        ("FR-R", "Risk Model", [
            ["FR-R-001", "Construct thermal features", "MUST"],
            ["FR-R-002", "Construct temporal features", "MUST"],
            ["FR-R-003", "Construct vulnerability features", "MUST"],
            ["FR-R-004", "Compute MRI in [0, 100]", "MUST"],
            ["FR-R-005", "Map MRI → band", "MUST"],
            ["FR-R-006", "Return top contributing factors", "MUST"],
            ["FR-R-007", "Pin risk_model_version string per run", "MUST"],
            ["FR-R-008", "Deterministic inference", "MUST"],
        ]),
        ("FR-A", "Agent", [
            ["FR-A-001", "Retrieves risk per ward", "MUST"],
            ["FR-A-002", "Retrieves forecast per ward", "MUST"],
            ["FR-A-003", "Retrieves vulnerability per ward", "MUST"],
            ["FR-A-004", "Retrieves resources", "MUST"],
            ["FR-A-005", "Reasons over ranked wards", "MUST"],
            ["FR-A-006", "Returns strictly structured JSON", "MUST"],
            ["FR-A-007", "Never invents weather or thermal values", "MUST"],
            ["FR-A-008", "Never claims mortality certainty", "MUST"],
            ["FR-A-009", "Suggests re-check time", "MUST"],
            ["FR-A-010", "Falls back to rule-based plan", "MUST"],
        ]),
        ("FR-B", "Backend", [
            ["FR-B-001", "Expose REST API", "MUST"],
            ["FR-B-002", "Run pipeline on a schedule", "MUST"],
            ["FR-B-003", "Persist all intermediate outputs", "MUST"],
            ["FR-B-004", "Serve dashboard frontend", "MUST"],
            ["FR-B-005", "Health/version endpoints", "MUST"],
            ["FR-B-006", "Stamp pipeline_run_id on responses", "MUST"],
            ["FR-B-007", "MOCK and LIVE modes", "MUST"],
        ]),
        ("FR-F", "Frontend", [
            ["FR-F-001", "Render Hyderabad ward choropleth map", "MUST"],
            ["FR-F-002", "Color wards by risk_band", "MUST"],
            ["FR-F-003", "Ward detail panel", "MUST"],
            ["FR-F-004", "3–5 day forecast timeline", "MUST"],
            ["FR-F-005", "Agent recommendation panel", "MUST"],
            ["FR-F-006", "Alert center", "MUST"],
            ["FR-F-007", "Display explanatory reasons", "MUST"],
            ["FR-F-008", "Never compute WBGT/UTCI on client", "MUST"],
        ]),
    ]:
        S.append(Paragraph(f"<b>{label} — {header}</b>", styles["subsection"]))
        S.append(styled_table(
            ["ID", "Requirement", "Priority"],
            rows, styles,
            col_widths=[25 * mm, 130 * mm, 20 * mm],
        ))
        S.append(Spacer(1, 2 * mm))

    # Chapter 7 — Non-Functional
    S.extend(chapter_header("07", "Non-Functional Requirements",
                            "MVP bar vs production bar.", styles))
    S.append(styled_table(
        ["Concern", "MVP Bar", "Production Bar"],
        [
            ["Performance", "p95 ward risk < 500 ms (cached)", "< 100 ms with horizontal scaling"],
            ["Reliability", "Ingestion retries 3× with backoff", "Multi-region, dead-letter queue"],
            ["Observability", "JSON logs with pipeline_run_id", "Distributed tracing, metrics"],
            ["Explainability", "Top-3 contributing factors", "Full SHAP / counterfactuals"],
            ["Reproducibility", "Pin physics/risk versions", "Immutable model artifacts"],
            ["Maintainability", "Single-command local run", "CI/CD, lint, type-check"],
            ["API consistency", "Stable JSON contracts", "OpenAPI 3.1 spec published"],
            ["Fault tolerance", "MOCK mode fallback", "Circuit breakers"],
            ["Security", "No PII; risk-only outputs", "RBAC, audit log"],
            ["Privacy", "Aggregate ward-level only", "No PHI ever stored"],
            ["Scalability", "One city", "Multi-city federation"],
            ["Testability", "Per-domain unit + integration", "90 %+ coverage"],
        ],
        styles,
        col_widths=[40 * mm, 60 * mm, 75 * mm],
    ))

    # Chapter 8 — Architecture
    S.extend(chapter_header("08", "Architecture",
                            "Six orthogonal domains, each owning one responsibility.", styles))
    S.append(Paragraph(
        "The system is split into six orthogonal domains. Each owns a single responsibility and exposes a "
        "stable contract. No team may modify another team's internal implementation without changing the "
        "contract.", styles["body"]))
    S.append(Paragraph("<b>End-to-end architecture</b>", styles["subsection"]))
    S.append(architecture_diagram(styles))
    S.append(Spacer(1, 4 * mm))
    S.append(Paragraph(
        "Arrows in the diagram are contracts. Every contract is a JSON schema defined in chapter 19. "
        "Edges are versioned: physics_version and risk_model_version travel with every payload.",
        styles["body"]))

    # Chapter 9 — Domain Ownership
    S.extend(chapter_header("09", "Domain Ownership",
                            "Clear ownership matrix; no cross-team modifications.", styles))
    S.append(styled_table(
        ["Domain", "Owner", "Inputs", "Outputs", "Tech"],
        [
            ["Weather + Spatial", "ML/DS", "Open-Meteo + ward GeoJSON", "ward_weather", "Python"],
            ["Physics", "ML/DS", "ward_weather", "ward_thermal_stress", "Python"],
            ["Risk", "ML/DS", "Thermal + vulnerability", "ward_mortality_risk", "Python/ONNX"],
            ["Agent", "Agent engineer", "Risk + context + tools", "decision_plan", "Python + LLM"],
            ["Backend", "Backend", "Domain outputs", "REST API, alerts", "Go"],
            ["Frontend", "Frontend", "Backend API", "Dashboard", "React"],
        ],
        styles,
        col_widths=[28 * mm, 30 * mm, 40 * mm, 35 * mm, 25 * mm],
    ))
    S.extend(architecture_box("Contract Rule",
                              "A team may consume another team's outputs but must not modify them. "
                              "To change a contract, both teams must agree.", styles))

    # Chapter 10 — Data Flow
    S.extend(chapter_header("10", "Data Flow",
                            "Step-by-step transformation pipeline.", styles))
    S.append(Paragraph(
        "1. Backend scheduler triggers a pipeline_run_id. 2. Weather domain fetches Open-Meteo. "
        "3. Spatial domain computes centroids and IDW. 4. Physics computes WBGT and UTCI. "
        "5. Risk computes MRI and band. 6. Agent reasons and proposes actions. "
        "7. Backend persists every stage with the run ID. 8. Frontend polls and renders.", styles["body"]))

    # Chapter 11 — Weather
    S.extend(chapter_header("11", "Weather System", "Open-Meteo ingestion and unit normalization.", styles))
    S.append(Paragraph(
        "Open-Meteo is the primary operational input. It is free, no API key, and provides hourly data for "
        "a 3–5 day horizon at ~9 km grid resolution.", styles["body"]))
    S.append(Paragraph(
        "<b>Required variables:</b> temperature_2m, relative_humidity_2m, wind_speed_10m, "
        "shortwave_radiation, direct_radiation, diffuse_radiation, surface_pressure, cloud_cover.",
        styles["body"]))
    S.extend(architecture_box("Unit Normalization",
                              "wind km/h × 0.2778 → m/s.  pressure hPa × 100 → Pa.  timezone Asia/Kolkata.",
                              styles))

    # Chapter 12 — Spatial
    S.extend(chapter_header("12", "Spatial System", "Ward polygons, centroids, IDW.", styles))
    S.append(Paragraph(
        "ward_id is computed as uuid5(NAMESPACE_URL, f\"{ward_name}|{ward_number}\"). Ward numbers are not "
        "unique across zones. The centroid is used to sample Open-Meteo; IDW is the documented fallback "
        "when raw grid data become available.", styles["body"]))
    S.append(Paragraph(
        "<b>IDW intuition:</b> nearby weather points influence a ward more than distant ones, with weight "
        "1/d². Far-away points fade quickly.", styles["body"]))

    # Chapter 13 — Physics
    S.extend(chapter_header("13", "Physics System", "WBGT (Liljegren) + UTCI + solar geometry.", styles))
    S.append(Paragraph(
        "WBGT blends natural wet-bulb, globe temperature, and shaded dry-bulb. Liljegren uses direct and "
        "diffuse radiation, solar zenith, humidity, wind, and pressure. If radiation is missing, the "
        "system falls back to a simplified WBGT and emits wbgt_quality=degraded.", styles["body"]))
    S.append(Paragraph(
        "UTCI is the equivalent air temperature at reference conditions producing the same physiological "
        "strain as the actual environment. Inputs: T, RH, v, Tmrt. For MVP we use an operational "
        "polynomial approximation pinned by physics_version.", styles["body"]))

    # Chapter 14 — ML / Risk
    S.extend(chapter_header("14", "ML / Risk System",
                            "Honest, transparent, vulnerability-aware risk indicator.", styles))
    S.append(Paragraph(
        "No validated mortality ground truth is available to the team. The MRI is therefore a "
        "physiologically grounded, vulnerability-aware heat-health risk indicator — NOT a clinical "
        "mortality prediction. We state this explicitly in the UI and PDFs.", styles["body"]))
    S.append(Paragraph("<b>MRI formula (MVP, transparent):</b>", styles["subsection"]))
    S.append(code_block(
        "MRI = 100 * (\n"
        "    0.35 * norm(peak_wbgt_next_72h, 25, 35)\n"
        "  + 0.20 * norm(utci_now, 26, 46)\n"
        "  + 0.15 * norm(persistence_days_high, 0, 5)\n"
        "  + 0.20 * norm(vulnerability_index, 0, 1)\n"
        "  + 0.10 * norm(-overnight_recovery_gap_c, -10, 0)\n"
        ")",
        styles,
    ))
    S.append(Paragraph("<b>Risk bands:</b>", styles["subsection"]))
    bands = [
        ("LOW", BAND_LOW), ("MODERATE", BAND_MODERATE), ("HIGH", BAND_HIGH),
        ("VERY HIGH", BAND_VHIGH), ("EXTREME", BAND_EXTREME),
    ]
    band_ranges = ["0–19", "20–39", "40–59", "60–79", "80–100"]
    band_table = []
    for idx, (name, color) in enumerate(bands):
        cell = Paragraph(f"<b><font color='white'>{name}</font></b>", styles["body_small"])
        band_table.append([Table([[cell]], colWidths=[28 * mm],
                                  style=TableStyle([
                                      ("BACKGROUND", (0, 0), (-1, -1), color),
                                      ("ALIGN", (0, 0), (-1, -1), "CENTER"),
                                  ])),
                           Paragraph(band_ranges[idx], styles["body_small"])])
    bt = Table(band_table, colWidths=[32 * mm, 30 * mm])
    S.append(bt)

    # Chapter 15 — Agent
    S.extend(chapter_header("15", "Agent System",
                            "Decision-support and orchestration layer; never replaces science.", styles))
    S.extend(decision_box("What the agent MAY do",
                          "Retrieve risk, forecast, vulnerability, resources. Compare wards. Prioritize. "
                          "Generate explanations, action plans, public advisories. Decide re-check cadence.",
                          styles))
    S.extend(danger_box("What the agent MUST NOT do",
                        "Invent weather or thermal values. Calculate WBGT or UTCI itself. "
                        "Fabricate medical facts or mortality statistics. Override validated outputs. "
                        "Claim mortality certainty.", styles))
    S.append(Paragraph(
        "<b>Agent tools:</b> get_ward_risk, get_forecast, get_thermal_stress, get_vulnerability, "
        "get_resources, get_previous_alerts, create_alert, generate_action_plan.",
        styles["body"]))
    S.append(Paragraph("<b>Agent loop:</b>", styles["subsection"]))
    S.append(Paragraph(
        "SENSE → PREDICT → ASSESS → REASON → ACT → MONITOR → REASSESS. "
        "Each stage is auditable and uses only deterministic tool outputs.", styles["body"]))
    S.append(Paragraph("<b>Strict output schema (excerpt):</b>", styles["subsection"]))
    S.append(code_block(
        "{\n"
        "  \"severity\": \"EXTREME\",\n"
        "  \"priority_wards\": [\"ward_001\", \"ward_017\"],\n"
        "  \"reasoning_summary\": \"...\",\n"
        "  \"key_factors\": [\"...\"],\n"
        "  \"recommended_actions\": [\n"
        "    { \"action\": \"Activate cooling center\", \"priority\": \"HIGH\", \"target\": \"ward_001\" }\n"
        "  ],\n"
        "  \"public_advisory\": \"...\",\n"
        "  \"recheck_interval_minutes\": 60\n"
        "}",
        styles,
    ))

    # Chapter 16 — Backend
    S.extend(chapter_header("16", "Backend System",
                            "Go service: HTTP API, persistence, orchestration, alerts.", styles))
    S.append(styled_table(
        ["Method", "Route", "Purpose", "Owner"],
        [
            ["GET", "/api/health", "Liveness", "Backend"],
            ["GET", "/api/version", "Versions", "Backend"],
            ["GET", "/api/wards", "List wards", "Backend"],
            ["GET", "/api/wards/{id}", "Ward detail", "Backend"],
            ["GET", "/api/wards/{id}/weather", "Weather", "Backend"],
            ["GET", "/api/wards/{id}/thermal", "Thermal stress", "Backend"],
            ["GET", "/api/wards/{id}/risk", "MRI + band", "Backend"],
            ["GET", "/api/forecast", "Citywide forecast", "Backend"],
            ["GET", "/api/alerts", "Active alerts", "Backend"],
            ["POST", "/api/agent/action-plan", "Agent recommendation", "Backend + Agent"],
            ["POST", "/api/pipeline/run", "Trigger full run", "Backend"],
        ],
        styles,
        col_widths=[18 * mm, 60 * mm, 65 * mm, 32 * mm],
    ))

    # Chapter 17 — Frontend
    S.extend(chapter_header("17", "Frontend System",
                            "GIS map, ward detail, forecast timeline, alerts, decision panel.", styles))
    S.append(Paragraph("<b>Five core screens:</b>", styles["subsection"]))
    S.append(Paragraph(
        "1. City Overview — choropleth map of wards. (3. Ward Detail — T, RH, v, rad, WBGT, UTCI, MRI, band. "
        "3. Forecast Timeline — 3–5 day risk evolution. 4. Alert Center — alerts with status. "
        "5. Decision Panel — agent explanation + actions.", styles["body"]))
    S.extend(danger_box("Frontend Constraint",
                        "Never compute WBGT/UTCI on the browser. Display physics_version and risk_model_version. "
                        "Show the disclaimer banner at all times.",
                        styles))

    # Chapter 18 — Database
    S.extend(chapter_header("18", "Database",
                            "ward_id is the stable UUID join key.", styles))
    S.append(styled_table(
        ["Table", "Purpose"],
        [
            ["ward_boundaries", "Polygon GeoJSON per ward"],
            ["ward_demographics", "Elderly, outdoor workers, housing, population"],
            ["raw_weather_staging", "Raw Open-Meteo payloads"],
            ["ward_weather", "Per-ward per-timestamp normalized weather"],
            ["ward_thermal_stress", "WBGT + UTCI per ward per timestamp"],
            ["ward_mortality_risk", "MRI + band + top factors per ward per run"],
            ["cooling_centers", "Resources per ward"],
            ["hospital_capacity", "Beds available per ward"],
            ["alerts", "Persisted alerts"],
            ["pipeline_runs", "Per-run stages and latency"],
            ["agent_decisions", "Agent plan per run per ward"],
        ],
        styles,
        col_widths=[55 * mm, 120 * mm],
    ))

    # Chapter 19 — API Contracts
    S.extend(chapter_header("19", "API Contracts",
                            "JSON schemas between every domain.", styles))
    S.append(Paragraph(
        "All endpoints return JSON. All responses include pipeline_run_id, physics_version, "
        "risk_model_version where applicable. Error envelope is documented in the SRS.",
        styles["body"]))
    S.append(Paragraph("<b>Example: GET /api/wards/{id}/risk</b>", styles["subsection"]))
    S.append(code_block(
        "{\n"
        "  \"ward_id\": \"ward_001\",\n"
        "  \"risk\": { \"score\": 82, \"band\": \"EXTREME\",\n"
        "             \"top_factors\": [\"Peak WBGT = 34.7 °C\",\n"
        "                              \"UTCI now = 45.1 °C\"] },\n"
        "  \"thermal\": { \"wbgt_c\": 33.8, \"utci_c\": 44.6 },\n"
        "  \"model_version\": \"risk-v1.0.0\"\n"
        "}",
        styles,
    ))

    # Chapter 20 — Agent Tool Contracts
    S.extend(chapter_header("20", "Agent Tool Contracts",
                            "Tool wrappers and strict output schema.", styles))
    S.append(styled_table(
        ["Tool", "Purpose"],
        [
            ["get_ward_risk", "Retrieve MRI + band + top factors"],
            ["get_forecast", "Retrieve forecast"],
            ["get_thermal_stress", "Retrieve WBGT, UTCI"],
            ["get_vulnerability", "Retrieve vulnerability features"],
            ["get_resources", "Retrieve cooling centers, hospitals"],
            ["get_previous_alerts", "Retrieve alert history"],
            ["create_alert", "Persist alert (side-effect)"],
            ["generate_action_plan", "Persist action plan (side-effect)"],
        ],
        styles,
        col_widths=[55 * mm, 120 * mm],
    ))
    S.extend(decision_box("Failure Behavior",
                          "If the LLM fails or times out, use the rule-based fallback plan and "
                          "set fallback_used=true. The plan is still persisted.",
                          styles))

    # Chapter 21 — Monorepo
    S.extend(chapter_header("21", "Monorepo Structure",
                            "Practical scaffold for parallel team work.", styles))
    S.append(code_block(
        "heatwave-early-warning/\n"
        "  docs/         # SRS, ARCHITECTURE, API_CONTRACTS, AGENT_SPEC, ...\n"
        "  backend/      # Go service: cmd/, internal/{api,services,...}\n"
        "  ml/           # Python: ingestion, spatial, physics, risk\n"
        "  agent/        # Python: tools, prompts, schemas, workflows\n"
        "  frontend/     # React + Vite + Leaflet\n"
        "  data/         # wards/, demographics/, fixtures/\n"
        "  scripts/      # dev and demo scripts\n"
        "  tools/        # PDF generator\n",
        styles,
    ))

    # Chapter 22 — Scaffold Routes
    S.extend(chapter_header("22", "Scaffold Routes",
                            "Backend endpoints already wired in the scaffold.", styles))
    S.append(styled_table(
        ["Method", "Route", "Service Called", "Tables Touched"],
        [
            ["GET", "/api/health", "Health", "—"],
            ["GET", "/api/version", "Version", "—"],
            ["GET", "/api/wards", "ListWards", "ward_boundaries"],
            ["GET", "/api/wards/{id}", "GetWard", "ward_boundaries, ward_demographics"],
            ["GET", "/api/wards/{id}/weather", "GetWardWeather", "ward_weather"],
            ["GET", "/api/wards/{id}/thermal", "GetWardThermal", "ward_thermal_stress"],
            ["GET", "/api/wards/{id}/risk", "GetWardRisk", "ward_mortality_risk"],
            ["GET", "/api/forecast", "GetForecast", "ward_thermal_stress, ward_mortality_risk"],
            ["GET", "/api/alerts", "ListAlerts", "alerts"],
            ["POST", "/api/agent/action-plan", "AgentActionPlan", "agent_decisions, alerts"],
            ["POST", "/api/pipeline/run", "RunOnce", "all"],
        ],
        styles,
        col_widths=[18 * mm, 55 * mm, 50 * mm, 50 * mm],
    ))

    # Chapter 23 — Env Config
    S.extend(chapter_header("23", "Environment Configuration",
                            "APP_MODE controls LIVE vs MOCK.", styles))
    S.append(code_block(
        "APP_MODE=MOCK\n"
        "PORT=8080\n"
        "OPEN_METEO_BASE=https://api.open-meteo.com/v1/forecast\n"
        "HYDERABAD_LAT=17.3850\n"
        "HYDERABAD_LON=78.4867\n"
        "DB_URL=postgresql://heatwave:heatwave@localhost:5432/heatwave\n"
        "AGENT_LLM_PROVIDER=mock\n"
        "LOG_LEVEL=info\n"
        "AUTO_RUN=false\n"
        "PIPELINE_INTERVAL=15m",
        styles,
    ))

    # Chapter 24 — Testing
    S.extend(chapter_header("24", "Testing Strategy",
                            "Per-layer tests; deterministic fixtures for reproducibility.", styles))
    S.append(styled_table(
        ["Layer", "Tests"],
        [
            ["Weather", "API parse, unit conversion, missing-field handling"],
            ["Spatial", "Geometry validation, centroid, IDW correctness"],
            ["Physics", "Reference WBGT/UTCI cases, edge inputs"],
            ["Risk", "Determinism, band boundaries"],
            ["Agent", "Tool selection, no hallucination, schema validity"],
            ["Backend", "Route tests, integration tests, error codes"],
            ["Frontend", "Map rendering, API failure handling, ward selection"],
        ],
        styles,
        col_widths=[28 * mm, 150 * mm],
    ))

    # Chapter 25 — Failure Handling
    S.extend(chapter_header("25", "Failure Handling",
                            "Fail-safe fallbacks at every layer.", styles))
    S.append(styled_table(
        ["Failure", "Impact", "Detection", "Fallback"],
        [
            ["Open-Meteo unavailable", "No new forecast", "HTTP timeout", "Last good response"],
            ["Missing radiation", "WBGT degraded", "Validation", "Simplified WBGT"],
            ["Invalid geometry", "Spatial failure", "Geometry check", "Exclude + log"],
            ["ML failure", "No MRI", "Inference error", "Thermal-only status"],
            ["Agent failure", "No recommendation", "LLM timeout", "Rule-based action plan"],
            ["Backend failure", "Dashboard unavailable", "Health check", "Retry / cached page"],
        ],
        styles,
        col_widths=[35 * mm, 35 * mm, 40 * mm, 70 * mm],
    ))

    # Chapter 26 — Observability
    S.extend(chapter_header("26", "Observability",
                            "Every log line is auditable.", styles))
    S.append(Paragraph(
        "Every log line includes: ts, level, service, pipeline_run_id, ward_id, stage, latency_ms, "
        "physics_version, risk_model_version. Reproducibility matters because alerts must be auditable.",
        styles["body"]))

    # Chapter 27 — Security & Safety
    S.extend(chapter_header("27", "Security & Safety",
                            "What this system must never do.", styles))
    S.extend(danger_box("Hard Limits",
                        "No clinical diagnosis. No individual-level prediction. No fabricated mortality. "
                        "All numeric outputs include provenance (physics_version, risk_model_version). "
                        "Agent never overwrites scientific outputs. Banner: decision-support prototype.",
                        styles))

    # Chapter 28 — 24h Plan
    S.extend(chapter_header("28", "24-Hour Execution Plan",
                            "Parallel streams; integration begins at hour 14.", styles))
    S.append(styled_table(
        ["Hour", "Stream A (ML)", "Stream B (Backend)", "Stream C (Frontend)", "Stream D (Agent)"],
        [
            ["0–2", "Contracts + fixtures", "Repo + DB", "Repo + UI shell", "Repo + tools"],
            ["2–5", "Weather + spatial", "Routes + schema", "Map base", "Tool schemas"],
            ["5–8", "WBGT + UTCI", "Orchestration", "API client", "Tool impl"],
            ["8–11", "Risk model", "Persist + serve", "Ward detail", "Prompt + LLM"],
            ["11–14", "Vulnerability", "Alerts", "Forecast timeline", "Structured output"],
            ["14–17", "Integration", "E2E API", "Wiring", "Action plan"],
            ["17–20", "Mock fixtures", "Health checks", "Polish", "Rule-based fallback"],
            ["20–22", "Tests", "Tests", "Tests", "Tests"],
            ["22–24", "Demo polish", "Demo polish", "Demo polish", "Demo polish"],
        ],
        styles,
        col_widths=[15 * mm, 40 * mm, 40 * mm, 40 * mm, 40 * mm],
    ))

    # Chapter 29 — Team Parallelization
    S.extend(chapter_header("29", "Team Parallelization Plan",
                            "Independent streams; integration near the end.", styles))
    S.append(Paragraph(
        "Four streams run in parallel. Integration begins at hour 14. ML/Physics, Backend, Frontend, "
        "and Agent each own their domain contracts and may not modify another team's outputs.",
        styles["body"]))

    # Chapter 30 — Domain Agent Prompts
    S.extend(chapter_header("30", "Domain-Specific Agent Prompts",
                            "Copy-paste prompts for each teammate's coding agent.", styles))
    for label, txt in [
        ("Prompt A — ML + Physics Agent", (
            "You are the ML + Physics engineer for the Heatwave Early Warning hackathon. "
            "Own weather ingestion, spatial downscaling, WBGT, UTCI, risk model. "
            "Stack: Python 3.11, NumPy, pandas, geopandas, shapely, scikit-learn. "
            "Files you OWN: ml/**, data/fixtures/**. Files you may READ but NOT modify: backend/**, agent/**, "
            "docs/API_CONTRACTS.md. Contracts: ward_weather, ward_thermal_stress, ward_mortality_risk. "
            "Tasks: Open-Meteo fetch, unit normalization, ward centroid + stable UUID, Liljegren WBGT with "
            "fallback, UTCI polynomial, vulnerability features, deterministic MRI + band + top factors. "
            "Forbidden: calculating decisions, calling LLM APIs, modifying backend contracts."
        )),
        ("Prompt B — Backend Agent", (
            "You are the Backend engineer. Own Go service, DB, pipeline orchestration, alerts. "
            "Stack: Go 1.22+, Gin or Echo. Files you OWN: backend/**. Tasks: scaffold module, migrations, "
            "routes per API_CONTRACTS.md, scheduler, persist all stages, MOCK mode, /api/health, /api/version. "
            "Forbidden: implementing WBGT/UTCI, implementing MRI, modifying agent prompts."
        )),
        ("Prompt C — Frontend Agent", (
            "You are the Frontend engineer. Own dashboard, GIS map, charts, ward detail, alert center. "
            "Stack: React 18, Vite, TypeScript, react-leaflet, recharts. Files you OWN: frontend/**. "
            "Tasks: choropleth map, ward detail panel, forecast timeline, alert center, decision panel, "
            "consume backend API, disclaimer banner visible, show physics_version and risk_model_version. "
            "Forbidden: computing WBGT/UTCI on browser, caching raw weather > 10 minutes."
        )),
        ("Prompt D — Decision Agent", (
            "You are the Decision Agent engineer. Own agent prompts, tools, schemas, workflows, action planning. "
            "Stack: Python 3.11, custom tool loop or LangGraph, OpenAI-compatible LLM. Files you OWN: agent/**. "
            "Contracts: output strictly follows schema in docs/AGENT_SPEC.md. Never modify scientific outputs. "
            "Tasks: implement all 8 tools, system prompt enforcing factual grounding + structured output, "
            "rule-based fallback, schema validation, agent_run_id on every decision. "
            "Forbidden: computing WBGT/UTCI, inventing weather, overriding risk model, claiming mortality certainty."
        )),
    ]:
        S.append(Paragraph(f"<b>{label}</b>", styles["subsection"]))
        S.append(code_block(txt, styles))

    # Chapter 31 — Demo Flow
    S.extend(chapter_header("31", "Demo Flow",
                            "3–5 minute hackathon walkthrough.", styles))
    demo_steps = [
        "1. Open Hyderabad choropleth. Show ward risk colors.",
        "2. Click a high-risk ward → show WBGT, UTCI, MRI, band, top factors.",
        "3. Switch to forecast timeline → show 3–5 day persistence.",
        "4. Click Ask Agent → query 'What should the municipality do?'",
        "5. Agent retrieves context and returns structured action plan.",
        "6. Show alert generation in Alert Center.",
        "7. Switch to MOCK mode → identical results with no internet.",
    ]
    for step in demo_steps:
        S.append(Paragraph(step, styles["body_small"]))

    # Chapter 32 — Differentiation
    S.extend(chapter_header("32", "Competitive Differentiation",
                            "Six layers of intelligence, one decision-support system.", styles))
    S.append(Paragraph(
        "The system combines six layers of intelligence: weather, spatial, physiological, vulnerability, "
        "temporal, decision. Most existing systems stop at weather. We go to decision with explicit, "
        "grounded reasoning. The agent is grounded in deterministic science; the agent never invents.",
        styles["body"]))
    S.extend(architecture_box("Sense → Predict → Reason → Act → Re-evaluate",
                              "Agentic loop grounded in physics, not free-form LLM prose.",
                              styles))

    # Chapter 33 — Limitations
    S.extend(chapter_header("33", "Limitations",
                            "What this system is not.", styles))
    S.extend(warning_box("Honest Limitations",
                         "Hackathon prototype. Not clinically validated. No ground-truth mortality dataset. "
                         "Open-Meteo ~9 km resolution may under-resolve UHI. No real-time weather station "
                         "telemetry. Agent decisions are advisory, not authoritative. MVP is read-only.",
                         styles))

    # Chapter 34 — Traceability
    S.extend(chapter_header("34", "Traceability Matrix",
                            "Requirement → component → implementation → contract → demo.", styles))
    S.append(styled_table(
        ["Requirement", "Component", "Implementation", "Contract", "Demo Evidence"],
        [
            ["Fetch weather", "ml/ingestion", "open_meteo.py", "raw_weather_staging", "Live fetch"],
            ["Downscale to ward", "ml/spatial", "wards.py", "ward_weather", "Map colors"],
            ["Compute WBGT", "ml/physics", "wbgt.py", "ward_thermal_stress", "Ward detail"],
            ["Compute UTCI", "ml/physics", "utci.py", "ward_thermal_stress", "Ward detail"],
            ["Vulnerability features", "ml/risk", "risk_model.py", "ward_demographics", "Top factors"],
            ["MRI", "ml/risk", "risk_model.py", "ward_mortality_risk", "Map + score"],
            ["Decision", "agent", "agent.py", "decision_plan", "Decision panel"],
            ["Alert", "backend", "alerts", "alerts table", "Alert center"],
            ["Map", "frontend", "Map.tsx", "/api/wards", "Live demo"],
            ["Disclaimer", "frontend", "Disclaimer.tsx", "static", "Visible banner"],
        ],
        styles,
        col_widths=[30 * mm, 30 * mm, 35 * mm, 40 * mm, 30 * mm],
    ))

    # Chapter 35 — Definition of Done
    S.extend(chapter_header("35", "Definition of Done",
                            "MVP is complete only when every item is checked.", styles))
    items = [
        "Hyderabad ward polygons load",
        "Stable ward IDs exist",
        "Open-Meteo data is fetched",
        "Weather units are normalized",
        "Ward-level weather is generated",
        "WBGT calculated (Liljegren with fallback)",
        "UTCI calculated",
        "3–5 day forecast exists",
        "Vulnerability features exist",
        "MRI generated",
        "Risk bands exist",
        "Agent retrieves risk",
        "Agent retrieves forecast",
        "Agent explains risk",
        "Agent generates structured actions",
        "Backend exposes APIs",
        "Frontend renders map",
        "Ward details work",
        "Alerts demonstrated",
        "MOCK mode works",
        "LIVE mode works",
        "End-to-end demo runs under 5 minutes",
    ]
    for i in items:
        S.append(Paragraph(f"[ ]  {i}", styles["body_small"]))

    S.append(Spacer(1, 6 * mm))
    S.extend(architecture_box("Closing Principle",
                              "Physics determines thermal stress. Risk estimates population-level risk. "
                              "Agent reasons about decisions. Backend orchestrates. Frontend communicates.",
                              styles))
    return S


# ──────────────────────────────────────────────────────────────────────────────
# Build PDF
# ──────────────────────────────────────────────────────────────────────────────

def main() -> None:
    styles = make_styles()
    doc = HEEWSDoc(str(OUTPUT), title="HEEWS SRS")
    story = build_story(styles)
    doc.build(story)
    print(f"Wrote {OUTPUT} ({OUTPUT.stat().st_size} bytes)")


if __name__ == "__main__":
    main()