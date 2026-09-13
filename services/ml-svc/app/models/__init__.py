"""services/ml-svc/app/models/

One directory per swappable model component (`storage`, `image_background`,
`image_lighting`, `vlm`, `embedding`, `reranking`, `handloom_texture`,
`voice_transcription`). Each owns its own Protocol, `Config`, and real/mock
implementations -- see any of those packages' `__init__.py` for the pattern.
Nothing here imports torch or transformers at module scope: in mock mode
those packages need not be installed.

There is no shared `Models` Protocol or `build()` any more -- `app/registry.py`
loads each component independently and `app/features/*.py` orchestrates
whichever ones a given RPC needs.
"""
