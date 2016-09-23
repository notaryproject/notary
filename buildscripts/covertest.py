#!/usr/bin/env python

"""
This module is used to run tests with full coverage-reports.

It's a way to provide accurate -coverpkg arguments to `go test`.

To run over all packages:
buildscripts/covertest.py --coverdir ".cover" --tags "pkcs11 mysql" --testopts="-v -race"

To run over some packages:
buildscripts/covertest.py --coverdir ".cover" --pkgs "X"
"""

from __future__ import print_function
from argparse import ArgumentParser
import os
import sys
import subprocess

BASE_PKG = "github.com/docker/notary"

def get_all_notary_pkgs(buildtags):
    out = subprocess.check_output(["go", "list", "-tags", buildtags, "./..."]).strip()
    return [
        p.strip() for p in out.split("\n")
        if not p.startswith(os.path.join(BASE_PKG, "vendor/"))
    ]

def get_coverprofile_filename(pkg, buildtags):
    if buildtags:
        buildtags = "." + buildtags.replace(' ', '.')
    return pkg.replace('/', '-').replace(' ', '_') + buildtags + ".coverage.txt"

def run_test_with_coverage(buildtags="", coverdir=".cover", pkgs=None, opts="", covermode="atomic"):
    """
    Run go test with coverage over the the given packages, with the following options
    """
    all_pkgs = get_all_notary_pkgs(buildtags)
    all_pkgs.sort()

    pkgs = pkgs or all_pkgs
    allpkg_string = ",".join(all_pkgs)

    base_cmd = ["go", "test", "-covermode", covermode, "-coverpkg", allpkg_string] + opts.split()
    if buildtags:
        base_cmd.extend(["-tags", buildtags])

    allpkg_string = ", ".join(all_pkgs)
    longest_pkg_len = max([len(pkg) for pkg in pkgs])

    for pkg in pkgs:
        cmd = base_cmd + ["-coverprofile", os.path.join(coverdir, get_coverprofile_filename(pkg, buildtags)), pkg]
        app = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        for line in app.stdout:
            # we are trying to generate coverage reports for everything in the base package, and some may not be
            # actually exercised in the test.  So ignore this particular warning.  Also, we want to pretty-print
            # the test success/failure results
            if not line.startswith("warning: no packages being tested depend on github.com/docker/notary"):
                line = line.replace(allpkg_string, "<all packages>").replace(pkg, pkg.ljust(longest_pkg_len))
                sys.stdout.write(line)

        app.wait()
        if app.returncode != 0:
            print("\n\nTests failed.\n")
            sys.exit(app.returncode)


def parseArgs(args=None):
    """
    CLI option parsing
    """
    parser = ArgumentParser()
    parser.add_argument("--coverdir", help="The coverage directory in which to put coverage files", default=".cover")
    parser.add_argument("--testopts", help="Options to pass for testing, such as -race or -v", default="")
    parser.add_argument("--pkgs", help="Packages to test specifically, otherwise we test all the packages", default="")
    parser.add_argument("--tags", help="Build tags to pass to go", default="")
    return parser.parse_args(args)

if __name__ == "__main__":
    args = parseArgs()
    pkgs = args.pkgs.strip().split()

    run_test_with_coverage(coverdir=args.coverdir, buildtags=args.tags, pkgs=pkgs, opts=args.testopts)
