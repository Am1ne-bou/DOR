#!/bin/bash

if [ $# -lt 3 ]; then
    echo "Usage: ./run_analysis.sh <output_dir> <log1> <log2> [log3] ..."
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUTPUT_DIR="$1"
shift

mkdir -p "$OUTPUT_DIR"

LOGS=()
for file in "$@"; do
    name=$(basename "$file" .log)
    grep "RESULT" "$file" > "$OUTPUT_DIR/${name}_results.log"
    LOGS+=("$OUTPUT_DIR/${name}_results.log")
done

python3 "$SCRIPT_DIR/analyze_bench.py" --output "$OUTPUT_DIR" "${LOGS[@]}"