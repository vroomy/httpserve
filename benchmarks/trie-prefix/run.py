#!/usr/bin/env python3
"""Run two prebuilt benchmark binaries serially, alternating order each round."""

import argparse
from datetime import datetime, timezone
import hashlib
import os
from pathlib import Path
import shlex
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("baseline", type=Path)
    parser.add_argument("candidate", type=Path)
    parser.add_argument("output", type=Path, help="new directory for results")
    parser.add_argument("--rounds", type=int, default=10)
    args = parser.parse_args()
    if args.rounds < 1:
        parser.error("--rounds must be positive")
    binaries = {
        "baseline": args.baseline.resolve(strict=True),
        "candidate": args.candidate.resolve(strict=True),
    }
    hashes = []
    for binary in binaries.values():
        with binary.open("rb") as source:
            hashes.append(hashlib.file_digest(source, "sha256").digest())
    if hashes[0] == hashes[1]:
        parser.error("baseline and candidate binaries are identical")
    suites = {
        "focused": (
            "^BenchmarkRouter(Stdlib|Allocations)$/"
            "^(URL=(Common|Short|Medium|Long)|DeepHit|DeepMiss|Fallback)$/"
            "^(Router=(ServeMux|Trie)|MatchInto|ServeHTTP)$",
            "1s",
        ),
        "matrix": ("^BenchmarkRouter(Match|ServeHTTP)$", "100ms"),
    }
    args.output.mkdir(parents=True, exist_ok=False)
    environment = {**os.environ, "GODEBUG": "", "GOEXPERIMENT": ""}
    with (args.output / "order.txt").open("w") as order_log:
        for suite, (selector, duration) in suites.items():
            for round_number in range(1, args.rounds + 1):
                order = ("baseline", "candidate")
                if round_number % 2 == 0:
                    order = tuple(reversed(order))
                for label in order:
                    command = [
                        str(binaries[label]), "-test.run", "^$",
                        "-test.bench", selector, "-test.benchmem",
                        "-test.benchtime", duration, "-test.count=1", "-test.cpu=1",
                    ]
                    stamp = datetime.now(timezone.utc).isoformat()
                    progress = f"{stamp} {suite} round={round_number} {label}"
                    print(progress, flush=True)
                    order_log.write(progress + "\n" + shlex.join(command) + "\n")
                    order_log.flush()
                    with (args.output / f"{suite}-{label}.txt").open("a") as raw:
                        subprocess.run(command, env=environment, stdout=raw,
                                       stderr=subprocess.STDOUT, check=True)
                    order_log.write("PASS\n")
                    order_log.flush()


if __name__ == "__main__":
    main()
