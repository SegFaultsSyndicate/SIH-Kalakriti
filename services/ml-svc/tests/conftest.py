"""services/ml-svc/tests/conftest.py"""

import sys
from pathlib import Path

# Tests import `app.*` directly, so the service root goes on the path once here
# rather than in every test file.
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
