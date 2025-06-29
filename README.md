# Unix Commands with Natural Language (uc)

A command line application that uses AI/LLM to interpret natural language commands and execute them as Unix utilities. Features an interactive mode with command history, OS detection, colorful output, and customizable system prompts.

## Features

- **Natural Language Processing**: Accepts commands in plain English
- **Multiple LLM Support**: Works with Ollama (default), OpenAI, and Google Gemini
- **JSON Configuration**: `.uc.json` configuration in your home directory
- **Interactive Mode**: REPL with command history and arrow key support
- **Colorful Interface**: Professional color-coded output using `github.com/fatih/color`
- **Loading Spinner**: Visual feedback during LLM processing with `github.com/briandowns/spinner`
- **Dry-Run Mode**: Preview commands without executing them (`-n` flag or `dryrun` command)
- **OS Detection**: Automatically detects your Unix OS for context-aware commands
- **Custom System Prompts**: Customize LLM behavior with your own prompt file
- **Command History**: Persistent history with `.uc_history` file
- **Smart Command Generation**: AI-powered Unix command generation
- **Robust Error Handling**: Clear feedback when commands can't be executed
- **Professional Output**: Clean, emoji-free interface suitable for enterprise environments

## Installation

```bash
# Clone and build
git clone https://github.com/sausheong/uc
cd uc
go build -o uc

# Optional: Install globally
sudo mv uc /usr/local/bin/
```

## Configuration

UC uses a JSON configuration file located at `~/.uc.json`. On first run, it will automatically create a default configuration:

```json
{
  "provider": "ollama",
  "ollama_url": "http://localhost:11434",
  "ollama_model": "llama3.2",
  "openai_key": "",
  "openai_model": "gpt-4.1-mini",
  "gemini_key": "",
  "gemini_model": "gemini-2.5-flash",
  "sys_prompt_file": "uc.prompts"
}
```

### Custom Configuration Path

You can specify a custom configuration file:

```bash
uc --config /path/to/custom.json "your command"
```

## Usage

### Interactive Mode (Default)

Run without arguments to start interactive mode:

```bash
uc
```

This starts a REPL (Read-Eval-Print Loop) with enhanced features:

```
uc (macOS 15.5) - Ollama (llama3.2)
Type your natural language commands below. Type 'exit' to exit, 'help' for help.

uc> list all files in the current directory
⠋ Generating command...
ls -la
total 64
drwxr-xr-x   8 user  staff   256 Dec 29 13:22 .
drwxr-xr-x  15 user  staff   480 Dec 29 13:20 ..
-rw-r--r--   1 user  staff    42 Dec 29 13:22 .gitignore
-rw-r--r--   1 user  staff   156 Dec 29 13:16 .uc.json
-rw-r--r--   1 user  staff  1234 Dec 29 13:22 README.md
-rw-r--r--   1 user  staff 15678 Dec 29 13:16 main.go

uc> dryrun
Dry-run mode enabled. Commands will be shown but not executed.

uc> find all Go files
⠋ Generating command...
find . -name "*.go" -type f
[DRY RUN] Command would execute: find . -name "*.go" -type f

uc> dryrun
Dry-run mode disabled. Commands will be executed normally.

uc> help
Help - Unix Commands in Natural Language

Commands:
  help          - Show this help message
  exit          - Exit the program
  dryrun        - Toggle dry-run mode (show commands without executing)

Examples of natural language commands:
  - list all files in the current directory
  - show me the contents of README.md
  - find all Go files in this directory
  - count the number of files in this directory

uc> exit
Goodbye!
```

**Interactive Mode Features:**
- **Command History**: Use arrow keys to navigate previous commands
- **Persistent History**: Commands saved to `.uc_history` file
- **Colorful Output**: Commands in cyan, errors in red, warnings in yellow
- **Loading Spinner**: Visual feedback while waiting for LLM responses
- **Dry-Run Toggle**: Type `dryrun` to toggle preview mode on/off
- **OS & LLM Info**: Shows your OS and LLM provider in the startup banner
- **Line Editing**: Full readline support with Ctrl+A, Ctrl+E, etc.

### Single Command Mode

Run with arguments for single command execution:

```bash
uc "list all files in the current directory"
uc "show me the contents of README.md"
uc "find all Go files"
```

### Dry-Run Mode

Preview commands without executing them:

```bash
# Using the -n flag
uc -n "delete all temporary files"
# Output: [DRY RUN] Command would execute: rm -f /tmp/*

# In interactive mode
uc> dryrun
Dry-run mode enabled. Commands will be shown but not executed.
uc> delete all log files
[DRY RUN] Command would execute: rm -f *.log
```

## Examples

```bash
# File operations
uc "list all files in the current directory"
uc "show me the contents of README.md"
uc "copy file1.txt to file2.txt"
uc "find all Go files in this directory"
uc "count the number of files in this directory"

# System information
uc "show me running processes"
uc "check disk usage"
uc "display memory usage"
uc "what's the current date and time"
uc "show system information"

# Text processing
uc "count lines in main.go"
uc "search for 'func' in all Go files"
uc "show the first 10 lines of the log file"
uc "replace 'old' with 'new' in file.txt"

# Network
uc "ping google.com"
uc "check if port 8080 is open"
uc "show network connections"

# Git operations
uc "show git status"
uc "list all git branches"
uc "show recent commits"
```

## Supported LLM Providers

| Provider | Configuration Fields | Notes |
|----------|---------------------|-------|
| **Ollama** | `ollama_url`, `ollama_model` | Default, runs locally, no API key needed |
| **OpenAI** | `openai_key`, `openai_model` | Requires API key |
| **Google Gemini** | `gemini_key`, `gemini_model` | Requires API key |

### Setting Up LLM Providers

**Ollama (Recommended)**
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model
ollama pull llama3.2

# UC will automatically use Ollama
```

**OpenAI**
```json
{
  "provider": "openai",
  "openai_key": "sk-your-api-key-here",
  "openai_model": "gpt-4.1-mini"
}
```

**Google Gemini**
```json
{
  "provider": "gemini",
  "gemini_key": "your-gemini-api-key-here",
  "gemini_model": "gemini-2.5-flash"
}
```

## Custom System Prompts

UC supports custom system prompts to modify LLM behavior. The system prompt file (`uc.prompts` by default) is automatically created on first run:

```
# Custom system prompts for UC
# Lines starting with # are comments and will be ignored
# Add your custom instructions below:

# Example: Always prefer verbose output
Always use verbose flags when available (like -v or --verbose).

# Example: Prefer specific tools
When listing files, prefer 'exa' over 'ls' if available.
```

You can customize the prompt file location in your `.uc.json`:
```json
{
  "sys_prompt_file": "/path/to/my/custom/prompts.txt"
}
```

## How It Works

1. **Input**: You provide a natural language command
2. **AI Processing**: The configured LLM interprets your request with OS context
3. **Command Generation**: AI generates the appropriate Unix command for your OS
4. **Execution**: The generated command is executed via shell (`/bin/sh -c`)
5. **Output**: Results are displayed with color-coded formatting

## Error Handling

UC provides clear, color-coded error messages with detailed feedback:

```bash
# Command generation error (red text)
Error generating command: connection refused
The command can't be executed - failed to generate Unix equivalent.

# Command execution error with stderr output (red text)
find . -printf "%s %p\n"
Error: find: -printf: unknown primary or operator
Command execution failed. See error details above.

# Dry-run mode preview
[DRY RUN] Command would execute: rm -rf /important/files
```

**Error Display Features:**
- **Colorful Output**: Errors in red, warnings in yellow, commands in cyan
- **Stderr Capture**: Always shows command stderr output when available
- **Context Information**: Clear indication of whether error occurred during generation or execution
- **Loading Feedback**: Spinner shows when waiting for LLM responses

**Common Issues:**
- LLM service unavailable (check if Ollama is running or API keys are valid)
- Generated command doesn't exist on your system (e.g., GNU vs BSD command differences)
- Ambiguous natural language input
- Invalid or expired API keys
- Platform-specific command incompatibilities (e.g., `find -printf` on macOS)

## OS Detection

UC automatically detects your operating system and includes this context in LLM prompts for more accurate command generation:

- **macOS**: Uses `sw_vers` for detailed version info
- **Linux**: Reads `/etc/os-release` for distribution details
- **Other Unix**: Falls back to `uname -sr`

## Files Created

- `~/.uc.json` - Main configuration file
- `uc.prompts` - Custom system prompts (in current directory)
- `.uc_history` - Command history for interactive mode (in current directory)

## Development

```bash
# Clone the repository
git clone https://github.com/sausheong/uc
cd uc

# Install dependencies
go mod tidy

# Build
go build -o uc

# Run tests
go test ./...

# Install globally (optional)
sudo mv uc /usr/local/bin/
```

### Dependencies

UC uses the following Go packages:
- `github.com/chzyer/readline` - Interactive command line with history
- `github.com/fatih/color` - Cross-platform colored terminal output
- `github.com/briandowns/spinner` - Terminal spinner for loading indication
- Standard library packages for HTTP, JSON, OS operations, and command execution

### Architecture

- **LLM Clients**: Modular design supporting multiple AI providers
- **Configuration**: JSON-based configuration with automatic creation
- **Command Execution**: Shell-based execution with proper stdout/stderr handling
- **Interactive Mode**: Readline-based REPL with persistent history
- **Error Handling**: Comprehensive error capture and colorful display

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
