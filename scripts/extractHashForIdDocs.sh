#!/bin/bash
# Check for 2 arguments
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <json_file> <search_directory>"
    exit 1
fi

JSON_FILE="$1"
SEARCH_DIR="$2"

# Validate inputs
if [ ! -f "$JSON_FILE" ]; then
    echo "Error: JSON file '$JSON_FILE' not found."
    exit 1
fi

if [ ! -d "$SEARCH_DIR" ]; then
    echo "Error: Directory '$SEARCH_DIR' not found."
    exit 1
fi
echo "# about to Convert JSON to lookup maps using jq with ${JSON_FILE}"
declare -A md5_map sha256_map
while IFS="|" read -r idd md5 sha256; do
    md5_map["$idd"]=$md5
    sha256_map["$idd"]=$sha256
done < <(jq -r '.[] | "\(.IDD)|\(.MD5)|\(.SHA256)"' "$JSON_FILE")

# Output files
MD5_OUTPUT="md5_output.txt"
SHA256_OUTPUT="sha256_output.txt"
ID_OUTPUT="id_output.txt"
# initialize output files
echo -n "" > "$MD5_OUTPUT"
echo -n "" > "$SHA256_OUTPUT"
echo "# will write md5 in ${MD5_OUTPUT} and sha256 in ${SHA256_OUTPUT}"
echo "# will now find and match files in : $SEARCH_DIR"
find "$SEARCH_DIR" -type f \( -iname '*.pdf' -o -iname '*.jpg' -o -iname '*.png' -o -iname '*.tif' \) | while read -r filepath; do
    filename=$(basename "$filepath")
    name="${filename%.*}"

    if [[ -n "${md5_map[$name]}" ]]; then
        echo "${md5_map[$name]}  $filepath" >> "$MD5_OUTPUT"
        echo "${sha256_map[$name]}  $filepath" >> "$SHA256_OUTPUT"
        echo "$name" >> "$ID_OUTPUT"
    fi
done
# next lines are for a special json formated array of records like this :
# {"IDD":"27201","NVE":"999999","ISP":"1","FIC":"Godoc%2Fmixte%2Fpdf%2F28000%2F27201.pdf","MD5":"4A874CDD01932645DE8CC23A083F8734","SHA256":"953b1b2220e4af679e370eae7f94ffbf02a20419b3387375a66a8c6c22bedafa","TYP":"15"},
echo "# extracting grep pattern from $ID_OUTPUT"
sed -E 's/^([0-9]+)$/{"IDD":"&"/' id_output.txt >id_pattern.txt
echo "# extracting info from ${JSON_FILE} based on existing files in path"
grep -f id_pattern.txt "$JSON_FILE" |sort -rn >files_info.json
jq 'map(.id = .IDD| .type = .TYP | if .NVE == "999999" then .version="" else .version=.NVE end |  del(.IDD, .ISP, .NVE, .TYP) )' files_info.json > files_info_formated.json

