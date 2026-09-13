"""services/ml-svc/app/models/translation/mock.py

A deterministic stand-in with the right shape, so the Go side can build
against a real wire contract with no weights on disk. Mock never rewrites
text content, only tags it with the target language -- so a do-not-translate
placeholder embedded in the input comes back byte-for-byte unchanged, the
same guarantee a real backend must independently provide for its own
reasons (see the package docstring).
"""

from __future__ import annotations


class MockTranslationModel:
    def translate(
        self,
        title: str,
        description: str,
        highlights: list[str],
        source_language: str,
        target_language: str,
    ) -> dict:
        tag = f"[{target_language}] "
        return {
            "title": tag + title,
            "description": tag + description,
            "highlights": [tag + h for h in highlights],
        }
