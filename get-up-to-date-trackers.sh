#!/bin/bash
# TODO: maybe validate each url, they should be valid but could have expired recently

set -euo pipefail # error handling, if anything fails error out

# updated every 24 hours, .txt file, newline seperated
url="https://raw.githubusercontent.com/ngosang/trackerslist/refs/heads/master/trackers_all.txt"
fileoutput="trackers.json"

# file looks like:
# {
#   udp: ['udp-tracker1', ...]
#   http: ['http-tracker1', ...]
# }

trackers=$(curl -fsSL $url)

# grep udp and http to seperate vars
udptrackers=$(printf '%s\n' "$trackers" | grep '^udp://') # grep for udp:// only for the start of the string
httptrackers=$(printf '%s\n' "$trackers" | grep '^https\?://') # grep for http(s):// only at the start of the string

# turn them into json lists
udptrackers=$(printf '%s' "$udptrackers" |
    jq -Rs 'split("\n") | map(select(length > 0))'
)

httptrackers=$(printf '%s' "$httptrackers" |
    jq -Rs 'split("\n") | map(select(length > 0))'
)

udpcount=$(printf '%s' "$udptrackers" | jq 'length')
httpcount=$(printf '%s' "$httptrackers" | jq 'length')

printf '[TRACKER-SCRIPT] Successfully found %d tracker(s) (%d UDP, %d HTTP)\n' \
    "$((udpcount + httpcount))" "$udpcount" "$httpcount"

# merge into a file (line 22 struct)
# save output to file
printf '{
    "udp": %s,
    "http": %s
}\n' "$udptrackers" "$httptrackers" > "$fileoutput"

printf "[TRACKER-SCRIPT] Saved tracker list to %s from root\n" "$fileoutput"
