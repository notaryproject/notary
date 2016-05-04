#!/usr/bin/env python

"""
Run basic notary client tests against a server
"""

from __future__ import print_function

import argparse
import inspect
import json
import os
from shutil import rmtree
from subprocess import CalledProcessError, PIPE, Popen, call
from tempfile import mkdtemp, mkstemp
from time import sleep, time
from uuid import uuid4

def reporoot():
    """
    Get the root of the git repo
    """
    return os.path.dirname(
        os.path.dirname(os.path.abspath(inspect.getfile(inspect.currentframe()))))

# Returns the reponame and server name
def parse_args(args=None):
    """
    Parses the command line args for this command
    """
    parser = argparse.ArgumentParser(description='Tests the notary client against a host')
    parser.add_argument(
        '-r', '--reponame', dest="reponame", type=str,
        help="The name of the repo - will be randomly generated if not provided")
    parser.add_argument(
        '-s', '--server', dest="server", type=str,
        help="Notary Server to connect to - defaults to https://notary-server:4443")
    parsed = parser.parse_args(args)

    return parsed.reponame, parsed.server

def cleanup(*paths):
    """
    Best effort removal the temporary paths, whether file or directory
    """
    for path in paths:
        try:
            os.remove(path)
        except OSError:
            pass
        else:
            continue

        try:
            rmtree(path)
        except OSError:
            pass

class Client(object):
    """
    Object that will run the notary client with the proper command lines
    """
    def __init__(self, notary_server):
        self.notary_server = notary_server

        binary = os.path.join(reporoot(), "bin", "notary")
        self.env = os.environ.copy()
        self.env.update({
            "NOTARY_ROOT_PASSPHRASE": "root_ponies",
            "NOTARY_TARGETS_PASSPHRASE": "targets_ponies",
            "NOTARY_SNAPSHOT_PASSPHRASE": "snapshot_ponies",
            "NOTARY_DELEGATION_PASSPHRASE": "user_ponies",
        })

        if notary_server is None:
            self.client = [binary, "-c", "cmd/notary/config.json"]
        else:
            self.client = [binary, "-s", notary_server]

    def run(self, args, trust_dir, stdinput=None):
        """
        Runs the notary client in a subprocess, and returns the output
        """
        command = self.client + ["-d", trust_dir] + list(args)
        print("$ " + " ".join(command))

        process = Popen(command, env=self.env, stdout=PIPE, stdin=PIPE)
        try:
            output, _ = process.communicate(stdinput)
        except Exception as ex:
            process.kill()
            process.wait()
            print("process failed: {0}".format(ex))
            raise
        retcode = process.poll()
        print(output)
        if retcode:
            raise CalledProcessError(retcode, command, output=output)
        return output


class Tester(object):
    """
    Thing that runs the test
    """
    def __init__(self, repo_name, client):
        self.repo_name = repo_name
        self.client = client
        self.dir = mkdtemp(suffix="_main")

    def basic_repo_test(self, tempfile, tempdir):
        """
        Initialize a repo, add a target, ensure the target is readable
        """
        print("---- Initializing a repo, adding a target, and pushing ----\n")
        self.client.run(["init", self.repo_name], self.dir)
        self.client.run(["add", self.repo_name, "basic_repo_test", tempfile], self.dir)
        self.client.run(["publish", self.repo_name], self.dir)

        print("---- Listing and validating basic repo test targets ----\n")
        targets1 = self.client.run(["list", self.repo_name], self.dir)
        targets2 = self.client.run(["list", self.repo_name], tempdir)

        assert targets1 == targets2
        assert "basic_repo_test" in targets1

        with open(tempfile) as target_file:
            text = target_file.read()

        self.client.run(["verify", self.repo_name, "basic_repo_test"], self.dir, stdinput=text)

    def add_delegation_test(self, tempfile, tempdir):
        """
        Add a delegation to the repo - assumes the repo has already been initialized
        """
        print("---- Rotating the snapshot key to server and adding a delegation ----\n")
        self.client.run(["key", "rotate", self.repo_name, "snapshot", "-r"], self.dir)
        self.client.run(
            ["delegation", "add", self.repo_name, "targets/releases",
             os.path.join(reporoot(), "fixtures", "secure.example.com.crt"), "--all-paths"],
            self.dir)
        self.client.run(["publish", self.repo_name], self.dir)

        print("---- Listing delegations ----\n")
        delegations1 = self.client.run(["delegation", "list", self.repo_name], self.dir)
        delegations2 = self.client.run(["delegation", "list", self.repo_name], tempdir)

        assert delegations1 == delegations2
        assert "targets/releases" in delegations1

        # add key to tempdir, publish target
        print("---- Publishing a target using a delegation ----\n")
        self.client.run(
            ["key", "import", os.path.join(reporoot(), "fixtures", "secure.example.com.key"),
             "-r", "targets/releases"],
            tempdir)
        self.client.run(
            ["add", self.repo_name, "add_delegation_test", tempfile, "-r", "targets/releases"],
            tempdir)
        self.client.run(["publish", self.repo_name], tempdir)

        print("---- Listing and validating delegation repo test targets ----\n")
        targets1 = self.client.run(["list", self.repo_name], self.dir)
        targets2 = self.client.run(["list", self.repo_name], tempdir)

        assert targets1 == targets2
        expected_target = [line for line in targets1.split("\n")
                           if line.strip().startswith("add_delegation_test") and
                           line.strip().endswith("targets/releases")]
        assert len(expected_target) == 1

    def root_rotation_test(self, tempfile, tempdir):
        """
        Test root rotation
        """
        print("---- Figuring out what the old keys are  ----\n")

        # update the tempdir
        self.client.run(["list", self.repo_name], tempdir)

        output = self.client.run(["key", "list"], self.dir)
        orig_root_key_info = [line.strip() for line in output.split("\n")
                              if line.strip().startswith('root')]
        assert len(orig_root_key_info) == 1

        # this should be replaced with notary info later
        with open(os.path.join(tempdir, "tuf", self.repo_name, "metadata", "root.json")) as root:
            root_json = json.load(root)
            old_root_num_keys = len(root_json["signed"]["keys"])
            old_root_certs = root_json["signed"]["roles"]["root"]["keyids"]
            assert len(old_root_certs) == 1

        print("---- Rotating root key  ----\n")
        # rotate root, check that we have a new key - this is interactive, so pass input
        self.client.run(["key", "rotate", self.repo_name, "root"], self.dir, stdinput="yes")
        output = self.client.run(["key", "list"], self.dir)
        new_root_key_info = [line.strip() for line in output.split("\n")
                             if line.strip().startswith('root') and
                             line.strip() != orig_root_key_info[0]]
        assert len(new_root_key_info) == 1

        # update temp dir and make sure we can download the update
        self.client.run(["list", self.repo_name], tempdir)
        with open(os.path.join(tempdir, "tuf", self.repo_name, "metadata", "root.json")) as root:
            root_json = json.load(root)
            assert len(root_json["signed"]["keys"]) == old_root_num_keys + 1
            root_certs = root_json["signed"]["roles"]["root"]["keyids"]
            assert len(root_certs) == 1
            assert root_certs != old_root_certs

        print("---- Ensuring we can still publish  ----\n")
        # make sure we can still publish from both repos
        self.client.run(
            ["key", "import", os.path.join(reporoot(), "fixtures", "secure.example.com.key"),
             "-r", "targets/releases"],
            tempdir)
        self.client.run(
            ["add", self.repo_name, "root_rotation_test_delegation_add", tempfile,
             "-r", "targets/releases"],
            tempdir)
        self.client.run(["publish", self.repo_name], tempdir)
        self.client.run(["add", self.repo_name, "root_rotation_test_targets_add", tempfile],
                        self.dir)
        self.client.run(["publish", self.repo_name], self.dir)

        targets1 = self.client.run(["list", self.repo_name], self.dir)
        targets2 = self.client.run(["list", self.repo_name], tempdir)

        assert targets1 == targets2
        lines = [line.strip() for line in targets1.split("\n")]
        expected_targets = [
            line for line in lines
            if (line.startswith("root_rotation_test_delegation_add") and
                line.endswith("targets/releases"))
            or (line.startswith("root_rotation_test_targets_add") and line.endswith("targets"))]
        assert len(expected_targets) == 2

    def run(self):
        """
        Run tests
        """
        for test_func in (self.basic_repo_test, self.add_delegation_test, self.root_rotation_test):
            _, tempfile = mkstemp()
            with open(tempfile, 'wb') as handle:
                handle.write(test_func.__name__ + "\n")

            tempdir = mkdtemp(suffix="_temp")

            try:
                test_func(tempfile, tempdir)
            except Exception:
                raise
            else:
                cleanup(tempfile, tempdir)

        cleanup(self.dir)

def wait_for_server(server, timeout_in_seconds):
    """
    Attempts to contact the server until it is up
    """
    command = ["curl", server]
    if server is None:
        server = "https://notary-server:4443"
        command = ["curl", "--cacert", os.path.join(reporoot(), "fixtures", "root-ca.crt"),
                   server]

    start = time()
    succeeded = False
    while time() <= start + timeout_in_seconds:
        proc = Popen(command, stderr=PIPE, stdin=PIPE)
        proc.communicate()
        if proc.poll():
            print("Waiting for {0} to be available.".format(server))
            sleep(1)
            continue
        else:
            succeeded = True
            break

    if not succeeded:
        raise Exception(
            "Could not connect to {0} after {2} seconds.".format(server, timeout_in_seconds))

def run():
    """
    Run the client tests
    """
    repo_name, server = parse_args()
    if not repo_name:
        repo_name = uuid4().hex
    if server is not None:
        server = server.lower().strip()

    if server in ("https://notary-server:4443", "https://notaryserver:4443", ""):
        server = None

    print("building a new client binary")
    call(['make', '-C', reporoot(), 'client'])
    print('---')

    username_passwd = ()
    if username is not None and username.strip():
        username = username.strip()
        password = getpass("password to server for user {0}: ".format(username))
        username_passwd = (username, password)

    wait_for_server(server, 30)

    Tester(repo_name, Client(server)).run()


if __name__ == "__main__":
    run()
