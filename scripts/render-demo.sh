#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

for tool in vhs ffmpeg ffprobe magick python3; do
  command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }
done

make build
mkdir -p dist docs/assets
demo_config_dir="$(mktemp -d)"
trap 'rm -rf "$demo_config_dir"' EXIT
export XDG_CONFIG_HOME="$demo_config_dir"
vhs validate scripts/demo.tape
vhs scripts/demo.tape

python3 - <<'PY'
import json
import subprocess

expected = {
    "examples/services.json": [
        {"name": "worker", "replicas": 8},
        {"name": "api", "replicas": 3},
    ],
    "examples/services-staging.json": [
        {"name": "scheduler", "replicas": 5},
        {"name": "gateway", "replicas": 2},
    ],
}
for source, result in expected.items():
    output = subprocess.check_output(["./bin/qbx", "history", "run", "1", source], text=True)
    assert json.loads(output) == result, (source, output)
print("Saved query produces the expected output on both JSON files")
PY

font="${QBX_DEMO_FONT:-/System/Library/Fonts/Supplemental/Arial.ttf}"
if [[ ! -f "$font" ]]; then
  echo "Recording labels need a font file; set QBX_DEMO_FONT (tried $font)" >&2
  exit 1
fi

labels=(
  "1  Open JSON and iterate services"
  "2  Filter language = python"
  "3  Select name and replicas"
  "4  Sort by replicas, descending"
  "5  Save the query"
  "6  Reuse it on another JSON file"
)
for i in "${!labels[@]}"; do
  magick -size 1280x64 xc:'#1e1e2e' \
    -font "$font" -pointsize 27 -fill '#cdd6f4' \
    -gravity center -annotate +0+0 "${labels[$i]}" "dist/demo-label-$i.png"
done

duration="$(ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 dist/demo-base.mp4)"
ffmpeg -hide_banner -loglevel error -y \
  -i dist/demo-base.mp4 \
  -loop 1 -i dist/demo-label-0.png \
  -loop 1 -i dist/demo-label-1.png \
  -loop 1 -i dist/demo-label-2.png \
  -loop 1 -i dist/demo-label-3.png \
  -loop 1 -i dist/demo-label-4.png \
  -loop 1 -i dist/demo-label-5.png \
  -filter_complex "[0:v]pad=iw:ih+64:0:64:color=0x1e1e2e[base];\
[base][1:v]overlay=0:0:enable='between(t,0,6)'[v1];\
[v1][2:v]overlay=0:0:enable='between(t,6,11)'[v2];\
[v2][3:v]overlay=0:0:enable='between(t,11,16)'[v3];\
[v3][4:v]overlay=0:0:enable='between(t,16,24)'[v4];\
[v4][5:v]overlay=0:0:enable='between(t,24,33)'[v5];\
[v5][6:v]overlay=0:0:enable='gte(t,33)'[out]" \
  -map '[out]' -an -t "$duration" -c:v libx264 -crf 20 -preset medium \
  -pix_fmt yuv420p -movflags +faststart docs/assets/demo.mp4

ffmpeg -hide_banner -loglevel error -y -i docs/assets/demo.mp4 \
  -filter_complex '[0:v]fps=8,split[a][b];[a]palettegen=max_colors=96:stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=3' \
  -loop 0 docs/assets/demo.gif

ffprobe -v error -show_entries format=duration,size -of default=noprint_wrappers=1 docs/assets/demo.mp4
du -h docs/assets/demo.gif docs/assets/demo.mp4
