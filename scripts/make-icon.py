"""Regenerate the checked-in Windows icon. Optional design dependency: Pillow.

The SVG uses the same geometry and colors. This script rasterizes those shapes
at 8x resolution before producing the standard ICO sizes. Runtime builds need
neither Python nor Pillow; use the checked-in icon/resource object.
"""
from pathlib import Path
from PIL import Image, ImageDraw

root = Path(__file__).resolve().parent.parent
scale = 8
image = Image.new("RGBA", (256 * scale, 256 * scale), (0, 0, 0, 0))
draw = ImageDraw.Draw(image)

def points(coords):
    # Rotate the A clockwise by 90 degrees around (128, 128), like a > prompt.
    return [((256 - y) * scale, x * scale) for x, y in coords]

draw.rounded_rectangle((8*scale, 8*scale, 248*scale, 248*scale), radius=52*scale, fill="#101b29")
draw.polygon(points([(55,181),(107,69),(135,69),(188,181),(157,181),(121,100),(85,181)]), fill="#54e0b5")
draw.polygon(points([(94,140),(148,140),(158,162),(84,162)]), fill="#f2f7fb")
draw.rounded_rectangle((150*scale,194*scale,196*scale,208*scale), radius=4*scale, fill="#ffb65c")
image = image.resize((512, 512), Image.Resampling.LANCZOS)
image.save(root / "assets/aslunix.png")
image.resize((32, 32), Image.Resampling.LANCZOS).save(root / "assets/aslunix-32.png")
image.save(root / "assets/aslunix.ico", sizes=[(n,n) for n in (16,20,24,32,40,48,64,128,256)])
print("Generated assets/aslunix.png and assets/aslunix.ico")
