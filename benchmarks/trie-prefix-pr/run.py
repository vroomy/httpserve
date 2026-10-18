#!/usr/bin/env python3
"""Compare prebuilt main and branch binaries using the existing stdlib fixtures."""

import argparse
from datetime import datetime, timezone
import hashlib
import os
from pathlib import Path
import shlex
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("main", type=Path)
    parser.add_argument("branch", type=Path)
    parser.add_argument("output", type=Path, help="new directory for raw samples")
    args = parser.parse_args()
    binaries = {"main": args.main.resolve(strict=True),
                "branch": args.branch.resolve(strict=True)}
    hashes = []
    for binary in binaries.values():
        with binary.open("rb") as source:
            hashes.append(hashlib.file_digest(source, "sha256").digest())
    if hashes[0] == hashes[1]:
        parser.error("main and branch binaries are identical")
    args.output.mkdir(parents=True, exist_ok=False)
    environment = {**os.environ, "GODEBUG": "", "GOEXPERIMENT": ""}
    with (args.output / "order.txt").open("w") as order_log:
        for round_number in range(1, 11):
            order = ("main", "branch") if round_number % 2 else ("branch", "main")
            for label in order:
                command = [str(binaries[label]), "-test.run", "^$",
                           "-test.bench", "^BenchmarkRouterStdlib$",
                           "-test.benchmem", "-test.benchtime=1s",
                           "-test.count=1", "-test.cpu=1"]
                stamp = datetime.now(timezone.utc).isoformat()
                progress = f"{stamp} round={round_number} {label}"
                print(progress, flush=True)
                order_log.write(progress + "\n" + shlex.join(command) + "\n")
                order_log.flush()
                with (args.output / f"{label}.txt").open("a") as raw:
                    subprocess.run(command, env=environment, stdout=raw,
                                   stderr=subprocess.STDOUT, check=True)
                order_log.write("PASS\n")
                order_log.flush()


if __name__ == "__main__":
    main()
