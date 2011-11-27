#!/usr/bin/env python

import os
import glob
import subprocess

def gccgo(*args):
	cmd = ["gccgo", "-g", "-O2", "-Wall", "-Ibuild"] + list(args)
	print " ".join(cmd)
	p = subprocess.Popen(cmd)
	retcode = p.wait()

	if retcode != 0:
		raise SystemExit(1)

def build_module(name, src_files, test_deps):
	print "***** Building %s" % name
	lib_file = os.path.join("build", name + ".o")
	gccgo("-c", "-o", lib_file, *src_files)

	print "***** Building %s_test" % name
	test_src = os.path.join("test", "%s_test.go" % name)
	test_prog = os.path.join("build", "%s_test" % name)
	gccgo("-o", test_prog, test_src, lib_file, *test_deps)

	return lib_file

def main():
	if not os.path.exists("build"):
		os.makedirs("build")

	argo_lib = build_module("argo", glob.glob("src/argo/*"), [])
	argo_kasabi_lib = build_module("argo_kasabi", ["src/apis/kasabi.go"], [argo_lib])

if __name__ == "__main__":
	main()