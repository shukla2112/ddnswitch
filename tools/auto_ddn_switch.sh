#!/bin/bash

# Function to switch DDN CLI version based on .ddn_cli_version file
function auto_switch_ddn() {
  # Check if .ddn_cli_version exists in the current directory
  if [[ -f ".ddn_cli_version" ]]; then
    local version=$(cat .ddn_cli_version | tr -d '[:space:]')
    
    # Only switch if version is not empty
    if [[ -n "$version" ]]; then
      # Check if we're already using this version
      local current_version=$(ddn version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)
      
      if [[ "$current_version" != "$version" ]]; then
        echo "📦 Switching DDN CLI to version $version"
        sudo ddnswitch "$version"
       #	> /dev/null
        
        if [[ $? -eq 0 ]]; then
          echo "✅ Successfully switched to DDN CLI $version"
        else
          echo "❌ Failed to switch to DDN CLI $version"
        fi
      else
        echo "✓ Already using DDN CLI $version"
      fi
    fi
  fi
}

# Function to be called when directory changes
function cd() {
  # Call the built-in cd command
  builtin cd "$@"
  
  # Run auto_switch_ddn after changing directory
  auto_switch_ddn
}

# Initialize on shell startup
auto_switch_ddn

echo "🔄 Auto DDN CLI version switching enabled"
echo "   Create a .ddn_cli_version file in your project directory"
echo "   with the desired version (e.g., v3.0.1) to auto-switch"
