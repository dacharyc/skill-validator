"""Extract data from a source file."""

import sys


def extract(path: str) -> str:
    """Read and return the contents of the given file."""
    with open(path) as f:
        return f.read()


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: extract.py <file>")
        sys.exit(1)
    print(extract(sys.argv[1]))
