#!/usr/bin/env python3
"""Save the current Go production import graph; arrows point to dependencies.

Run from any directory. Graphviz (`dot`) is needed only for SVG rendering.
`go list -e` keeps local import discovery usable when an external module is
unavailable; dependency-resolution errors are recorded, not treated as a
successful compilation. Standard-library and external-module edges are omitted.
"""

import argparse
import json
from pathlib import Path
import shutil
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tags", default="", help="Go build tags (default: native)")
    parser.add_argument("--output-dir", type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    output = args.output_dir or root / "doc"
    output.mkdir(parents=True, exist_ok=True)
    command = ["go", "list", "-e", "-json"]
    if args.tags:
        command += ["-tags", args.tags]
    result = subprocess.run(command + ["./..."], cwd=root / "pkg",
                            text=True, capture_output=True, check=True)
    decoder = json.JSONDecoder()
    remaining = result.stdout.lstrip()
    packages = []
    while remaining:
        package, end = decoder.raw_decode(remaining)
        packages.append(package)
        remaining = remaining[end:].lstrip()
    module = "github.com/glycerine/g3/pkg"
    names = {p["ImportPath"] for p in packages}
    short = lambda path: path.removeprefix(module + "/") if path != module else "pkg"
    nodes = []
    unresolved = set()
    for package in packages:
        if package.get("Error"):
            raise RuntimeError(package["Error"]["Err"])
        for error in package.get("DepsErrors", []):
            stack = error.get("ImportStack", [])
            unresolved.add(stack[-1] if stack else error["Err"])
        nodes.append({
            "package": short(package["ImportPath"]),
            "imports": sorted(short(p) for p in package.get("Imports", []) if p in names),
            "production_files": sorted(package.get("GoFiles", [])),
        })
    nodes.sort(key=lambda p: p["package"])
    snapshot = {"build_tags": args.tags or "native", "edge_direction": "consumer -> dependency",
                "scope": "local production imports; excludes test, standard-library and external edges",
                "unresolved_external_dependencies": sorted(unresolved), "packages": nodes}
    (output / "package_dependencies.json").write_text(json.dumps(snapshot, indent=2) + "\n")

    for stem, selected in [
        ("package_dependencies", nodes),
        ("util_dependencies", [p for p in nodes if p["package"] == "arena"
                               or p["package"].startswith("util/")]),
    ]:
        selected_names = {p["package"] for p in selected}
        lines = ["digraph packages {", "  rankdir=LR;", "  graph [bgcolor=white, pad=0.25];",
                 '  node [shape=box, fontname="sans-serif", fontsize=11];',
                 '  edge [color="#7a8793"];',
                 '  label="Go production imports: consumer -> dependency";', "  labelloc=t;"]
        for package in selected:
            name = package["package"]
            color = "#c9e8cb" if name == "arena" else "#dceafa" if name.startswith("util/") else "#eeeeee"
            lines.append(f"  {json.dumps(name)} [style=filled, fillcolor={json.dumps(color)}];")
            for dependency in package["imports"]:
                if dependency in selected_names:
                    lines.append(f"  {json.dumps(name)} -> {json.dumps(dependency)};")
        lines.append("}")
        dot = output / (stem + ".dot")
        dot.write_text("\n".join(lines) + "\n")
        if shutil.which("dot"):
            subprocess.run(["dot", "-Tsvg", str(dot), "-o", str(output / (stem + ".svg"))], check=True)
    print(f"Saved {len(nodes)} packages to {output}")
    if unresolved:
        print("Import discovery only; external dependencies unresolved: " + ", ".join(sorted(unresolved)))


if __name__ == "__main__":
    main()
