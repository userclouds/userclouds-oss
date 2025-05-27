#!/usr/bin/env bash

if [[ -t 1 ]]; then
  C_WHITE=$(tput setaf 7)
  C_LIGHTBLUE=$(tput setaf 6)
  C_GREEN=$(tput setaf 2)
  C_RED=$(tput setaf 1)
  C_BOLD=$(tput bold)
  C_RESET=$(tput sgr0)
else
  C_WHITE=""
  C_LIGHTBLUE=""
  C_GREEN=""
  C_RED=""
  C_BOLD=""
  C_RESET=""
fi

function debug() {
  echo "${C_BOLD}${C_WHITE}... ${C_LIGHTBLUE}$*${C_RESET}"
}

function info() {
  echo "${C_BOLD}${C_WHITE}>>> ${C_GREEN}$*${C_RESET}"
}

function error() {
  echo "${C_BOLD}${C_WHITE}!!! ${C_RED}$*${C_RESET}"
}
