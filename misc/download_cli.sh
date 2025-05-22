#!/usr/bin/env bash

# Some helpful functions
yell() { echo -e "${RED}FAILED> $* ${NC}" >&2; }
die() { yell "$*"; exit 1; }
try() { "$@" || die "failed executing: $*"; }
log() { echo -e "--> $*"; }

# Colors for colorizing
RED='\033[0;31m'
GREEN='\033[0;32m'
PURPLE='\033[0;35m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m'

INSTALL_PATH=${INSTALL_PATH:-"/usr/local/bin"}
NEED_SUDO=0

function maybe_sudo() {
    if [[ "$NEED_SUDO" == '1' ]]; then
        sudo "$@"
    else
        "$@"
    fi
}

# check for curl
hasCurl=$(which curl)
if [ "$?" = "1" ]; then
    die "You need to install curl to use this script."
fi

log "${GREEN}Installing Hasura DDN CLI"
log "${NC}"

log "Selecting version..."

# Install latest by default, overrideable by user via VERSION env var
version=${VERSION:-latest}

log "Selected version: $version"

log "${YELLOW}"
log NOTE: Install a specific version of the CLI by using VERSION variable
log 'curl -L https://graphql-engine-cdn.hasura.io/ddn/cli/v4/get.sh | VERSION=v1.0.0 bash'
log "${NC}"

# check for existing ddn installation
hasCli=$(which ddn)
if [ "$?" = "0" ]; then
    log ""
    log "${GREEN}You already have the DDN cli at '${hasCli}'${NC}"
    export n=3
    log "${YELLOW}Downloading again in $n seconds... Press Ctrl+C to cancel.${NC}"
    log ""
    sleep $n
fi

# get platform and arch
platform='unknown'
unamestr=$(uname)
if [[ "$unamestr" == 'Linux' ]]; then
    platform='linux'
elif [[ "$unamestr" == 'Darwin' ]]; then
    platform='darwin'
fi

if [[ "$platform" == 'unknown' ]]; then
    die "Unknown OS platform"
fi

arch='unknown'
archstr=$(uname -m)
if [[ "$archstr" == 'x86_64' ]]; then
    arch='amd64'
elif [[ "$archstr" == 'arm64' ]] || [[ "$archstr" == 'aarch64' ]]; then
    arch='arm64'
else
    # TODO; check if the messaging is OK
    die "prebuilt binaries for $(arch) architecture not available, please reach out to us via https://hasura.io/contact-us"
fi

# some variables
suffix="-${platform}-${arch}"
targetFile="/tmp/cli-ddn$suffix"

if [ -e $targetFile ]; then
    rm $targetFile
fi

log "${PURPLE}Downloading DDN for $platform-$arch to ${targetFile}${NC}"
url=https://graphql-engine-cdn.hasura.io/ddn/cli/v4/$version/cli-ddn$suffix

try curl -L# -f -o $targetFile "$url"
try chmod +x $targetFile

log "${GREEN}Download complete!${NC}"

# check for sudo
needSudo=$(mkdir -p "${INSTALL_PATH}" && touch "${INSTALL_PATH}/.ddninstall" &> /dev/null)
if [[ "$?" == "1" ]]; then
    NEED_SUDO=1
fi
rm "${INSTALL_PATH}/.ddninstall" &> /dev/null

if [[ "$NEED_SUDO" == '1' ]]; then
    log
    log "${YELLOW}Path '$INSTALL_PATH' requires root access to write."
    log "${YELLOW}This script will attempt to execute the move command with sudo.${NC}"
    log "${YELLOW}Are you ok with that? (y/N)${NC}"
    read a
    if [[ $a == "Y" || $a == "y" || $a = "" ]]; then
        log
    else
        log
        log "  ${BLUE}sudo mv $targetFile ${INSTALL_PATH}/ddn${NC}"
        log
        die "Please move the binary manually using the command above."
    fi
fi

log "Moving cli from $targetFile to ${INSTALL_PATH}"

try maybe_sudo mkdir -p "${INSTALL_PATH}"
try maybe_sudo mv $targetFile "${INSTALL_PATH}/ddn"

log
log "${GREEN}DDN cli installed to ${INSTALL_PATH}${NC}"
log

if [ -e $targetFile ]; then
    rm $targetFile
fi

ddn doctor

if ! $(echo "$PATH" | grep -q "$INSTALL_PATH"); then
    log
    log "${YELLOW}$INSTALL_PATH not found in \$PATH, you might need to add it${NC}"
    log 
fi