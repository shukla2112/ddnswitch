# DDNSwitch

A command-line utility to easily switch between different versions of the Hasura DDN CLI, similar to `tfswitch` for Terraform.

Deepwiki docs link - https://deepwiki.com/shukla2112/ddnswitch/2.1-version-management-engine

## Features

- 🔄 **Easy Version Switching**: Switch between DDN CLI versions with a simple command
- 📋 **Interactive Selection**: Choose from available versions using an interactive menu
- 📦 **Automatic Installation**: Downloads and installs DDN CLI versions on demand
- 🗂️ **Version Management**: List, install, and uninstall specific versions
- 🔗 **Smart Linking**: Automatically creates symlinks or copies binaries to PATH
- 🖥️ **Cross-Platform**: Works on Linux, macOS, and Windows
- ⚡ **Fast**: Caches downloaded versions locally for quick switching
- 🔒 **Permission-Aware**: Falls back to user directories when system directories aren't writable

## Installation

### From Source

```bash
git clone https://github.com/yourusername/ddnswitch.git
cd ddnswitch
make build
make install-user  # Installs to ~/bin
```

### Using Go Install

```bash
go install github.com/yourusername/ddnswitch@latest
```

## Usage

### Interactive Mode

Simply run `ddnswitch` without arguments to see an interactive menu:

```bash
ddnswitch
```

To include pre-release versions:

```bash
ddnswitch --pre
```

This will:
1. Fetch available DDN CLI versions
2. Show an interactive selection menu
3. Download and install the selected version (if not already cached)
4. Switch your active DDN CLI to the selected version

### Direct Version Selection

Switch to a specific version directly:

```bash
ddnswitch v3.0.1
```

### List Available Versions

```bash
ddnswitch list
```

To include pre-release versions:

```bash
ddnswitch list --pre
```

### Install a Specific Version

```bash
ddnswitch install v3.0.1
```

### Show Current Version

```bash
ddnswitch current
```

### Uninstall a Version

```bash
ddnswitch uninstall v3.0.1
```

### Show DDNSwitch Version

```bash
ddnswitch version
```

### Debug Mode

For troubleshooting, you can enable debug mode to see detailed logging:

```bash
ddnswitch --debug list
```

This will show additional information about:
- Network requests
- Cache operations
- Version detection
- Installation steps

## How It Works

1. **Version Discovery**: DDNSwitch fetches available DDN CLI versions
2. **Local Storage**: Downloaded versions are stored in `~/.ddnswitch/` directory
3. **Path Management**: Creates symlinks or copies the selected binary to a directory in your PATH
4. **Caching**: Once downloaded, versions are cached locally for fast switching
5. **Permission Handling**: Automatically falls back to user directories when system directories aren't writable

## Directory Structure

```
~/.ddnswitch/
├── v3.0.1/
│   └── ddn
├── v3.0.0/
│   └── ddn
└── v2.9.0/
    └── ddn
```

## Requirements

- Go 1.21+ (for building from source)
- Internet connection (for downloading DDN CLI versions)
- Write access to a directory in your PATH (or ~/bin will be created)

## Configuration

DDNSwitch works out of the box with no configuration required. It will:

1. Create `~/.ddnswitch/` directory for storing DDN CLI versions
2. Try to create symlinks in the first writable directory found in your PATH
3. Fall back to `~/bin` if no suitable directory is found in PATH

## Platform Support

DDNSwitch supports the following platforms:

- **Linux**: x86_64 (ARM-based Linux is not supported)
- **macOS**: x86_64 (Intel), arm64 (Apple Silicon)
- **Windows**: x86_64

## Building

### Prerequisites

```bash
go mod tidy
```

### Build for Current Platform

```bash
make build
```

### Build for All Platforms

```bash
make build-all
```

### Available Make Targets

- `make build` - Build for current platform
- `make build-all` - Build for all supported platforms
- `make install` - Install to `/usr/local/bin`
- `make install-user` - Install to `~/bin`
- `make clean` - Remove build artifacts
- `make test` - Run tests
- `make check` - Run all code checks (format, vet, test)

## Troubleshooting

### DDNSwitch not found after installation

Make sure the installation directory is in your PATH:

```bash
# For ~/bin installation
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Permission denied when creating symlinks

DDNSwitch will automatically try to find a writable directory in your PATH. If it can't find one, it will:
1. Create `~/bin` directory
2. Install the binary there
3. Notify you to add this directory to your PATH

### No compatible binary found

This means the DDN CLI doesn't have a release for your platform. DDNSwitch supports:
- Linux (x86_64)
- macOS (Intel and Apple Silicon)
- Windows (x86_64)

Note that ARM-based Linux systems are not currently supported.

### Network issues

DDNSwitch requires internet access to:
- Fetch the list of available versions
- Download DDN CLI releases

If you're behind a proxy or firewall, ensure your Go environment is configured correctly.

### Slow version fetching

DDNSwitch caches the list of available versions for 1 hour to improve performance. The first fetch might take a few seconds, but subsequent operations will be much faster.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [tfswitch](https://github.com/warrensbox/terraform-switcher) for Terraform
- Built for the [Hasura DDN CLI](https://hasura.io/docs/3.0/reference/cli/installation)

## Related Projects

- [tfswitch](https://github.com/warrensbox/terraform-switcher) - Terraform version switcher
- [g](https://github.com/voidint/g) - Go version manager
- [nvm](https://github.com/nvm-sh/nvm) - Node.js version manager
