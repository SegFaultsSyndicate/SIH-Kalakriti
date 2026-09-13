"""services/ml-svc/tests/test_config.py"""

from app.config import load_craft_vocab


def test_load_craft_vocab_reads_pipe_separated_columns(tmp_path):
    csv_path = tmp_path / "crafts.csv"
    csv_path.write_text(
        "code,display_name,parent_code,gi_registration_no,techniques,materials\n"
        "ajrakh-block-printing,Ajrakh Block Printing,,,"
        "hand-block-printing|resist-dyeing,cotton|natural-indigo\n"
    )
    vocab = load_craft_vocab(csv_path)
    assert vocab["ajrakh-block-printing"]["materials"] == ["cotton", "natural-indigo"]
    assert vocab["ajrakh-block-printing"]["techniques"] == ["hand-block-printing", "resist-dyeing"]


def test_load_craft_vocab_is_empty_for_a_plain_code_list(tmp_path):
    # load_craft_allowlist's other accepted shape: no techniques/materials
    # columns at all. Should not raise, and should yield no vocabulary.
    csv_path = tmp_path / "codes.txt"
    csv_path.write_text("ajrakh-block-printing\nbagru-block-printing\n")
    assert load_craft_vocab(csv_path) == {}


def test_load_craft_vocab_missing_file_is_not_fatal(tmp_path):
    assert load_craft_vocab(tmp_path / "nope.csv") == {}
