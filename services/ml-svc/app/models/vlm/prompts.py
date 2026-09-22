"""services/ml-svc/app/models/vlm/prompts.py

Prompt text shared by every VLM backend (`real.py`'s transformers path,
`llamacpp.py`'s llama.cpp path) -- kept in one place so the two backends are
asked the same question, not two subtly different ones that drift apart.
"""

from __future__ import annotations

EXTRACT_PROMPT = """You are cataloguing an Indian handicraft for a marketplace.
Answer with JSON only, matching this schema exactly:
{{"craft_id": str, "material": str, "technique": str, "colours": [str], "motifs": [str],
 "confidence": {{"craft_id": float, "material": float, "technique": float, "colours": float, "motifs": float}}}}

craft_id MUST be one of these exact codes, or "" if none of them fit:
{allowlist}

The artisan says this is: {declared}
The artisan adds: {hint}{vocab_hint}
Do not guess beyond what you can see. Use "" or [] where you are unsure."""

POLISH_PROMPT = """Rewrite this marketplace product description with a warmer,
more persuasive tone. State only the facts already in it — do not add any
material, technique, colour, motif, region, age, award or other claim that is
not already present in the text below.

{template}

Write in {language}, under {max_chars} characters. Reply with the rewritten
description only: no JSON, no preamble, no quotation marks."""


def technique_prompt(craft_id: str, claimed: str) -> str:
    return (
        f"These frames show a craftsperson working on a {craft_id.replace('-', ' ')} piece.\n"
        f'Answer JSON only: {{"observed": str, "matches": bool, "confidence": float, '
        f'"explanation": str}}\n'
        f"observed is the technique you actually see. matches is whether it is "
        f"{claimed.replace('-', ' ')}. One sentence of explanation."
    )
