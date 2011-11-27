#!/usr/bin/env python

import sys
import time
import subprocess

N = 100

def run_benchmark(cmd):
	cmd += " > /dev/null 2>&1"
	start = time.time()

	for i in xrange(N):
		print "\r%d/%d" % (i + 1, N),
		sys.stdout.flush()
		p = subprocess.Popen(cmd, shell=True)
		p.wait()
	
	end = time.time()
	print

	return (end - start) / float(N)

def benchmark_result(secs):
	ms = secs * 1000.0
	us = secs * 1000.0

	return "%.3fs, %.3fms, %.3fus" % (secs, ms, us)

if __name__ == "__main__":
	print "============ Prerequisites ============"
	print "***** Compiling benchmark-argo"
	p = subprocess.Popen(["gccgo", "-g", "-O2", "-Wall", "-I", "../build", "-o", "benchmark-argo", "benchmark-argo.go", "../build/argo.o"])
	p.wait()

	print "============ Benchmarks ============"
	print "***** benchmark-argo"
	b_argo = run_benchmark("./benchmark-argo")

	print "***** benchmark-rdflib"
	b_rdflib = run_benchmark("python benchmark-rdflib.py")

	print "============ Results ============"
	print "benchmark-argo: %s" % benchmark_result(b_argo)
	print "benchmark-rdflib: %s" % benchmark_result(b_rdflib)