#!/usr/bin/env bash

main() {
  version="$1"
  binaries_dir="$2"

  change_log_file="./CHANGELOG.md"
  version=`printf "## $version" | tr -d 'v'`
  version_prefix="## [0-9]{1,2}\."
  start=0
  CHANGE_LOG=""
  while IFS= read line; do
    if [[ $line == *"$version"* ]]; then
      start=1
      continue
    fi
    if [[ $line =~ $version_prefix ]] && [ $start == 1 ]; then
      break;
    fi
    if [ $start == 1 ]; then
      CHANGE_LOG+="$line\n"
    fi
  done < ${change_log_file}

  LINUX_X86_64_BIN_SUM="$(checksum "$binaries_dir/dummy-blockchain-x86_64-unknown-linux-gnu")"

  OUTPUT="$(cat <<-END
## Changelog
${CHANGE_LOG}
## Checksums
|Assets | Checksum (sha256)|
|-|-|
|thegarii-x86_64-unknown-linux-gnu | ${LINUX_X86_64_BIN_SUM}|
END
)"

  echo -e "${OUTPUT}"
}

checksum() {
  echo $(sha256sum $@ | awk '{print $1}')
}

main $@