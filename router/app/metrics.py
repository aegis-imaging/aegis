"""Tiny Prometheus-text-format metrics exposer.

We avoid the `prometheus_client` dep to keep the router image small. The
metrics are simple counters + a duration histogram with fixed buckets — enough
for throughput benchmarking on a research install ("how many studies/minute
can my MacBook router push through?"). For production grade observability,
swap this for prometheus_client + a proper sidecar.
"""

from __future__ import annotations

import threading
from collections import defaultdict
from typing import Iterable


# Fixed buckets in seconds — cover the realistic spread for a study going
# through tag-de-id + optional defacing (which is the slow stage).
_DURATION_BUCKETS = (0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0, 600.0, 1800.0)


class Metrics:
    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._counters: dict[str, dict[tuple[tuple[str, str], ...], float]] = defaultdict(dict)
        self._histograms: dict[
            str, dict[tuple[tuple[str, str], ...], dict[str, float]]
        ] = defaultdict(dict)

    def inc(self, name: str, labels: dict[str, str] | None = None, value: float = 1.0) -> None:
        key = self._labels_key(labels)
        with self._lock:
            self._counters[name][key] = self._counters[name].get(key, 0.0) + value

    def observe(self, name: str, value: float, labels: dict[str, str] | None = None) -> None:
        key = self._labels_key(labels)
        with self._lock:
            hist = self._histograms[name].setdefault(key, self._fresh_histogram())
            for b in _DURATION_BUCKETS:
                if value <= b:
                    hist[f"le_{b}"] += 1
            hist["count"] += 1
            hist["sum"] += value

    def render(self) -> str:
        """Render in Prometheus text format."""
        lines: list[str] = []
        with self._lock:
            for name, series in sorted(self._counters.items()):
                lines.append(f"# TYPE {name} counter")
                for label_key, val in sorted(series.items()):
                    lines.append(f"{name}{self._fmt_labels(label_key)} {_fmt_num(val)}")
            for name, series in sorted(self._histograms.items()):
                lines.append(f"# TYPE {name} histogram")
                for label_key, hist in sorted(series.items()):
                    base = self._fmt_labels(label_key)
                    for b in _DURATION_BUCKETS:
                        lines.append(
                            f'{name}_bucket{self._fmt_labels(label_key, extra=("le", str(b)))} {_fmt_num(hist[f"le_{b}"])}'
                        )
                    lines.append(
                        f'{name}_bucket{self._fmt_labels(label_key, extra=("le", "+Inf"))} {_fmt_num(hist["count"])}'
                    )
                    lines.append(f"{name}_count{base} {_fmt_num(hist['count'])}")
                    lines.append(f"{name}_sum{base} {_fmt_num(hist['sum'])}")
        return "\n".join(lines) + "\n"

    @staticmethod
    def _fresh_histogram() -> dict[str, float]:
        h: dict[str, float] = {"count": 0.0, "sum": 0.0}
        for b in _DURATION_BUCKETS:
            h[f"le_{b}"] = 0.0
        return h

    @staticmethod
    def _labels_key(labels: dict[str, str] | None) -> tuple[tuple[str, str], ...]:
        if not labels:
            return ()
        return tuple(sorted(labels.items()))

    @staticmethod
    def _fmt_labels(key: Iterable[tuple[str, str]], extra: tuple[str, str] | None = None) -> str:
        items = list(key)
        if extra is not None:
            items = items + [extra]
        if not items:
            return ""
        pairs = ",".join(f'{k}="{_escape(v)}"' for k, v in items)
        return "{" + pairs + "}"


def _fmt_num(v: float) -> str:
    if v == int(v):
        return str(int(v))
    return f"{v:g}"


def _escape(s: str) -> str:
    return s.replace("\\", "\\\\").replace('"', '\\"').replace("\n", "\\n")


# Single process-wide instance — created in main.py at startup.
_global: Metrics | None = None


def init() -> Metrics:
    global _global
    if _global is None:
        _global = Metrics()
    return _global


def get() -> Metrics:
    if _global is None:
        return init()
    return _global
