#!/bin/bash
set -e
set -u

ENV=local
PARAMETER=

BASE_DIR=$(dirname $0)
SCRIPT_PATH="$( cd "${BASE_DIR}" && pwd -P )"

load_env(){
  ENV_FILE="${SCRIPT_PATH}/env/${ENV}.env"
  if test -f "${ENV_FILE}"; then
      source "${ENV_FILE}"
  fi
}
load_env

exit_err() {
  echo "ERROR: ${1}" >&2
  exit 1
}

# Usage: -h, --help
# Description: Show help text
option_help() {
  printf "Usage: %s [options...] COMMAND <parameter> \n\n" "${0}"
  printf "Default command: --help\n\n"

  echo "Options:"
  grep -e '^[[:space:]]*# Usage:' -e '^[[:space:]]*# Description:' -e '^option_.*()[[:space:]]*{' "${0}" | while read -r usage; read -r description; read -r option; do
    if [[ ! "${usage}" =~ Usage ]] || [[ ! "${description}" =~ Description ]] || [[ ! "${option}" =~ ^option_ ]]; then
      exit_err "Error generating help text."
    fi
    printf " %-32s %s\n" "${usage##"# Usage: "}" "${description##"# Description: "}"
  done

  printf "\n"
  echo "Commands:"
  grep -e '^[[:space:]]*# Command Usage:' -e '^[[:space:]]*# Command Description:' -e '^command_.*()[[:space:]]*{' "${0}" | while read -r usage; read -r description; read -r command; do
    if [[ ! "${usage}" =~ Usage ]] || [[ ! "${description}" =~ Description ]] || [[ ! "${command}" =~ ^command_ ]]; then
      exit_err "Error generating help text."
    fi
    printf " %-32s %s\n" "${usage##"# Command Usage: "}" "${description##"# Command Description: "}"
  done
}

# Command Usage: tag
# Command Description: Git release with version
command_tag() {
  git tag ${PARAMETER} && git push origin ${PARAMETER}
}

# Command Usage: test
# Command Description: go test
command_test() {
  go test ./...
}

# Command Usage: coverage
# Command Description: go test coverage
command_coverage() {
  go test -coverprofile=coverage.out -covermode=atomic ./...
  go tool cover -func=coverage.out
}

# Command Usage: bump
# Command Description: Bump version
command_bump() {
  # 定义文件路径
  MANIFEST_FILE="manifest.json"
  VERCURR_FILE="./pkg/version/current.go"

  # 检查文件是否存在
  if [[ ! -f "$MANIFEST_FILE" ]]; then
    echo "Error: $MANIFEST_FILE does not exist."
    exit 1
  fi

  if [[ ! -f "$VERCURR_FILE" ]]; then
    echo "Error: $VERCURR_FILE does not exist."
    exit 1
  fi

  # 读取当前的 version
  current_version=$(jq -r '.version' "$MANIFEST_FILE")

  # 分割 version 字符串，提取主版本号、中版本号、修订号
  IFS='.' read -r major minor patch <<< "$current_version"

  # 对最后一位修订号进行递增
  new_patch=$((patch + 1))

  # 生成新的版本号
  new_version="$major.$minor.$new_patch"

  # 更新 manifest.json 文件中的 version
  jq --arg new_version "$new_version" '.version = $new_version' "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"

  # 更新 vercurr.go 文件的内容
  cat > "$VERCURR_FILE" << EOL
package version

var CurrentVersion = Version{
    Major:      $major,
    Minor:      $minor,
    PatchLevel: $new_patch,
    Suffix:     "",
}
EOL

  echo "Version bumped to $new_version and vercurr.go updated"
}



# Command Usage: rn
# Command Description: Get release notes
command_release_notes() {
  # 获取最新的 release notes
  release_notes=$(git log $(git describe --tags --abbrev=0)..HEAD --oneline)

  # 打印 release notes 到屏幕上
  echo "### Release Notes:"
  echo "$release_notes"

  # 保存 release notes 到文件，并添加到 Git 暂存区
  echo "$release_notes" > release-notes.md
}

check_msg() {
  printf "\xE2\x9C\x94 ${1}\n"
}

main() {
  [[ -z "${@}" ]] && eval set -- "--help"

  local theCommand=

  set_command() {
    [[ -z "${theCommand}" ]] || exit_err "Only one command at a time!"
    theCommand="${1}"
  }

  while (( ${#} )); do
    case "${1}" in

      --help|-h)
        option_help
        exit 0
        ;;

      tag|test|coverage|bump|rn)
        set_command "${1}"
        ;;

      *)
        PARAMETER="${1}"
        ;;
    esac

    shift 1
  done

  [[ ! -z "${theCommand}" ]] || exit_err "Command not found!"

  case "${theCommand}" in
    run) command_run;;
    test) command_test;;
    coverage) command_coverage;;
    clean) command_clean;;
    tag) command_tag;;
    bump) command_bump;;
    rn) command_release_notes;;

    *) option_help; exit 1;;
  esac
}

main "${@-}"