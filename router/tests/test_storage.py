from app import storage


def test_layout_creates_raw_and_deid(tmp_data_dir):
    p = storage.layout(tmp_data_dir, "1.2.3.4")
    assert p.raw.is_dir()
    assert p.deid.is_dir()
    assert p.manifest.parent.is_dir()


def test_next_raw_index_increments(tmp_data_dir):
    p = storage.layout(tmp_data_dir, "1.2.3.4")
    assert storage.next_raw_index(p) == 0
    storage.write_raw_bytes(p, 0, b"hello")
    assert storage.next_raw_index(p) == 1
    storage.write_raw_bytes(p, 1, b"world")
    assert storage.next_raw_index(p) == 2


def test_list_files_sorted(tmp_data_dir):
    p = storage.layout(tmp_data_dir, "1.2.3.4")
    # write out-of-order to confirm sort
    storage.write_raw_bytes(p, 2, b"c")
    storage.write_raw_bytes(p, 0, b"a")
    storage.write_raw_bytes(p, 1, b"b")
    files = storage.list_raw_files(p)
    assert [f.name for f in files] == ["0.dcm", "1.dcm", "2.dcm"]


def test_manifest_roundtrip(tmp_data_dir):
    p = storage.layout(tmp_data_dir, "1.2.3.4")
    storage.write_manifest(p, {"hello": "world", "count": 3})
    assert storage.read_manifest(p) == {"hello": "world", "count": 3}


def test_read_manifest_missing_returns_empty(tmp_data_dir):
    p = storage.layout(tmp_data_dir, "1.2.3.4")
    assert storage.read_manifest(p) == {}
