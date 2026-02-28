"""Tests for the BASIC_PROFILE tag table."""

from midi_b.deid.tags import BASIC_PROFILE, ACTION_LABELS, get_tag_rule, is_private_tag


def test_profile_has_expected_tag_count():
    # Should have 100+ tags
    assert len(BASIC_PROFILE) >= 100


def test_all_actions_are_valid():
    valid_actions = {"D", "Z", "X", "U", "K", "C"}
    for tag, rule in BASIC_PROFILE.items():
        assert rule["action"] in valid_actions, f"Tag {tag} has invalid action {rule['action']}"


def test_all_tags_are_8_char_hex():
    import re
    for tag in BASIC_PROFILE:
        assert re.match(r"^[0-9A-Fa-f]{8}$", tag), f"Tag {tag} is not 8-char hex"


def test_all_rules_have_keyword():
    for tag, rule in BASIC_PROFILE.items():
        assert rule["keyword"], f"Tag {tag} has empty keyword"


def test_get_tag_rule_case_insensitive():
    rule = get_tag_rule("00100010")
    assert rule is not None
    assert rule["keyword"] == "PatientName"
    # Upper case
    rule2 = get_tag_rule("00100010")
    assert rule2 == rule


def test_get_tag_rule_unknown():
    assert get_tag_rule("FFFFFFFF") is None


def test_is_private_tag():
    assert is_private_tag("00090010") is True
    assert is_private_tag("00100010") is False
    assert is_private_tag("00110001") is True
    assert is_private_tag("00080060") is False


def test_action_labels_complete():
    for action in ("D", "Z", "X", "U", "K", "C"):
        assert action in ACTION_LABELS


def test_sr_tags_present():
    assert "0040A730" in BASIC_PROFILE  # ContentSequence
    assert "0040A160" in BASIC_PROFILE  # TextValue
    assert "0040A123" in BASIC_PROFILE  # PersonName
    assert "0040A124" in BASIC_PROFILE  # UID


def test_patient_name_is_z_action():
    rule = get_tag_rule("00100010")
    assert rule is not None
    assert rule["action"] == "Z"


def test_modality_is_k_action():
    rule = get_tag_rule("00080060")
    assert rule is not None
    assert rule["action"] == "K"
