from app import metrics


def test_counter_increments():
    m = metrics.Metrics()
    m.inc("aegis_router_studies_received_total", {"source": "dimse"})
    m.inc("aegis_router_studies_received_total", {"source": "dimse"})
    m.inc("aegis_router_studies_received_total", {"source": "web_upload"})
    out = m.render()
    assert 'aegis_router_studies_received_total{source="dimse"} 2' in out
    assert 'aegis_router_studies_received_total{source="web_upload"} 1' in out


def test_counter_no_labels():
    m = metrics.Metrics()
    m.inc("aegis_router_studies_shipped_total")
    m.inc("aegis_router_studies_shipped_total")
    out = m.render()
    assert "aegis_router_studies_shipped_total 2" in out


def test_histogram_records_buckets_and_sum():
    m = metrics.Metrics()
    for v in (0.3, 1.5, 4.0, 12.0):
        m.observe("aegis_router_pipeline_duration_seconds", v, {"outcome": "shipped"})
    out = m.render()
    # 0.3 fits in every bucket from 0.5 up
    assert 'aegis_router_pipeline_duration_seconds_bucket{outcome="shipped",le="0.5"} 1' in out
    # 0.3 and 1.5 fit in <=2.0 (so 2 in the 2.0 bucket)
    assert 'aegis_router_pipeline_duration_seconds_bucket{outcome="shipped",le="2"} 2' in out
    assert 'aegis_router_pipeline_duration_seconds_count{outcome="shipped"} 4' in out
    assert 'aegis_router_pipeline_duration_seconds_sum{outcome="shipped"} 17.8' in out


def test_label_escaping():
    m = metrics.Metrics()
    m.inc("test_metric", {"path": 'with "quotes" and \\slash'})
    out = m.render()
    assert r'path="with \"quotes\" and \\slash"' in out


def test_global_singleton():
    a = metrics.init()
    b = metrics.get()
    assert a is b
