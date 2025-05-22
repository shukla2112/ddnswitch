# Auto DDN CLI Version Switcher

A simple bash utility that automatically switches to the correct DDN CLI version when you change directories, based on a `.ddn_cli_version` file.

## Overview

When working on multiple Hasura DDN projects that require different versions of the DDN CLI, manually switching between versions can be tedious. This utility solves that problem by:

1. Detecting when you change directories
2. Checking for a `.ddn_cli_version` file
3. Automatically switching to the specified DDN CLI version

## Prerequisites

- Bash shell
- [ddnswitch](https://github.com/yourusername/ddnswitch) installed and in your PATH
- DDN CLI installed

## Installation

1. Clone this repository or download the `auto_ddn_switch.sh` script

2. Make the script executable:
   ```bash
   chmod +x auto_ddn_switch.sh
   ```

3. Add the following line to your `~/.bashrc` or `~/.zshrc` file:
   ```bash
   source /path/to/auto_ddn_switch.sh
   ```

4. Reload your shell configuration:
   ```bash
   source ~/.bashrc  # or source ~/.zshrc
   ```

## Usage

1. Create a `.ddn_cli_version` file in your project directory:
   ```bash
   echo "v3.0.1" > /path/to/your/project/.ddn_cli_version
   ```

2. When you `cd` into that directory, the script will automatically switch to the specified DDN CLI version:
   ```bash
   cd /path/to/your/project
   # Output: 📦 Switching DDN CLI to version v3.0.1
   # Output: ✅ Successfully switched to DDN CLI v3.0.1
   ```

3. Different projects can specify different versions:
   ```
   project1/.ddn_cli_version  # contains "v3.0.1"
   project2/.ddn_cli_version  # contains "v2.9.0"
   ```

## How It Works

The script works by:

1. **Overriding the `cd` command**: It creates a custom `cd` function that calls the built-in `cd` command and then runs our version-checking logic.

2. **Version detection**: When you change directories, it checks for a `.ddn_cli_version` file and reads the version from it.

3. **Version comparison**: It compares the current DDN CLI version with the target version to avoid unnecessary switching.

4. **Version switching**: If needed, it uses `ddnswitch` to change to the specified version.

5. **Feedback**: It provides visual feedback about the switching process.

## Example Output

```
$ cd project1
📦 Switching DDN CLI to version v3.0.1
✅ Successfully switched to DDN CLI v3.0.1

$ cd project2
📦 Switching DDN CLI to version v2.9.0
✅ Successfully switched to DDN CLI v2.9.0

$ cd project1
✓ Already using DDN CLI v3.0.1
```

## Troubleshooting

- **Script not working after installation**: Make sure you've sourced your `.bashrc` or `.zshrc` file after adding the source line.

- **Version not switching**: Ensure `ddnswitch` is installed and in your PATH. Check that the version specified in `.ddn_cli_version` is available.

- **Error messages**: If you see "Failed to switch to DDN CLI version", check that the version exists and is available through `ddnswitch list`.

## License

MIT