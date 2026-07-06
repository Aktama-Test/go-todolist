# Perf-test image for T-9442: several incompressible layers so the `load`
# (pull into the local docker daemon) is where time is actually spent.
FROM ubuntu:24.04
RUN dd if=/dev/urandom of=/l1.bin bs=1M count=100
RUN dd if=/dev/urandom of=/l2.bin bs=1M count=100
RUN dd if=/dev/urandom of=/l3.bin bs=1M count=100
RUN dd if=/dev/urandom of=/l4.bin bs=1M count=100
