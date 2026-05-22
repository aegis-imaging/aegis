from dimse_core.storage import StudyLayout


def test_layout_creates_dirs(tmp_data_dir):
    layout = StudyLayout(tmp_data_dir, "1.2.3").ensure()
    assert layout.raw_dir.is_dir()
    assert layout.clean_dir.is_dir()


def test_next_raw_index_increments(tmp_data_dir):
    layout = StudyLayout(tmp_data_dir, "1.2.3").ensure()
    assert layout.next_raw_index() == 0
    layout.write_raw(0, b"a")
    assert layout.next_raw_index() == 1
    layout.write_raw(1, b"b")
    assert layout.next_raw_index() == 2


def test_list_raw_sorted(tmp_data_dir):
    layout = StudyLayout(tmp_data_dir, "1.2.3").ensure()
    layout.write_raw(2, b"c")
    layout.write_raw(0, b"a")
    layout.write_raw(1, b"b")
    files = layout.list_raw()
    assert [f.name for f in files] == ["0.dcm", "1.dcm", "2.dcm"]


def test_size_bytes(tmp_data_dir):
    layout = StudyLayout(tmp_data_dir, "1.2.3").ensure()
    layout.write_raw(0, b"hello")
    layout.write_raw(1, b"world!")
    assert layout.raw_size_bytes() == 11
    assert layout.clean_size_bytes() == 0
