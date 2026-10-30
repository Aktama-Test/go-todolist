import pytest


@pytest.mark.benchmark
def test_sum():
    assert sum(range(10_000)) == 49_995_000
