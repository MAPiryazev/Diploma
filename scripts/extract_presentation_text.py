from __future__ import annotations

import re
import sys
import xml.etree.ElementTree as ET
import zipfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_PRESENTATION = ROOT / "docs" / "презентация — копия.pptx"
TEXT_NS = "{http://schemas.openxmlformats.org/drawingml/2006/main}t"


def slide_number(name: str) -> int:
    match = re.search(r"slide(\d+)\.xml$", name)
    if not match:
        raise ValueError(f"cannot parse slide number from {name}")
    return int(match.group(1))


def main() -> None:
    path = Path(sys.argv[1]) if len(sys.argv) > 1 else DEFAULT_PRESENTATION
    if not path.is_absolute():
        path = ROOT / path

    with zipfile.ZipFile(path) as archive:
        slides = sorted(
            [
                name
                for name in archive.namelist()
                if re.match(r"ppt/slides/slide\d+\.xml$", name)
            ],
            key=slide_number,
        )
        print(f"slides: {len(slides)}")
        for slide in slides:
            root = ET.fromstring(archive.read(slide))
            texts = [
                node.text.strip()
                for node in root.iter(TEXT_NS)
                if node.text and node.text.strip()
            ]
            print(f"\n--- slide {slide_number(slide)} ---")
            print("\n".join(texts))


if __name__ == "__main__":
    main()
