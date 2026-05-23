from app.config import RouterConfig


def test_load_defaults(base_env):
    cfg = RouterConfig.load()
    assert cfg.site_name == "test-site"
    assert cfg.dimse_ae_title == "TEST_AE"
    assert cfg.dimse_port == 0
    assert cfg.midi_b_salt == "test-salt"
    assert cfg.enabled_sidecars() == {}
    assert cfg.cloud_forwarding_configured() is False


def test_enabled_sidecars(base_env, monkeypatch):
    monkeypatch.setenv("DEFACING_SERVICE_URL", "http://localhost:8081")
    monkeypatch.setenv("PHI_DETECTION_SERVICE_URL", "http://localhost:8082")
    cfg = RouterConfig.load()
    assert cfg.enabled_sidecars() == {
        "defacing": "http://localhost:8081",
        "phi_detection": "http://localhost:8082",
    }


def test_cloud_forwarding_needs_all_three(base_env, monkeypatch, tmp_path):
    cert = tmp_path / "client.crt"
    key = tmp_path / "client.key"
    cert.write_text("x")
    key.write_text("x")

    monkeypatch.setenv("CLOUD_RECEIVER_URL", "https://cloud.example.com")
    cfg = RouterConfig.load()
    assert cfg.cloud_forwarding_configured() is False  # missing cert/key

    monkeypatch.setenv("CLOUD_CLIENT_CERT", str(cert))
    monkeypatch.setenv("CLOUD_CLIENT_KEY", str(key))
    cfg = RouterConfig.load()
    assert cfg.cloud_forwarding_configured() is True


def test_retained_tags_parsed_as_tuple(base_env, monkeypatch):
    monkeypatch.setenv("MIDI_B_RETAINED_TAGS", "ProtocolName, BodyPartExamined ,, StudyDescription")
    cfg = RouterConfig.load()
    assert cfg.midi_b_retained_tags == ("ProtocolName", "BodyPartExamined", "StudyDescription")
