package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// Constants
const (
	// Default configuration values
	DefaultProvider    = "ollama"
	DefaultOllamaURL   = "http://localhost:11434"
	DefaultOllamaModel = "llama3.2"
	DefaultOpenAIModel = "gpt-4.1-mini"
	DefaultGeminiModel = "gemini-2.5-flash"
	DefaultConfigFile  = ".uc.json"
	DefaultHistoryFile = ".uc_history"
	DefaultPromptFile  = "uc.prompts"

	// Interactive commands
	CmdHelp   = "help"
	CmdExit   = "exit"
	CmdDryRun = "dryrun"

	// Prompts
	NormalPrompt = "uc> "

	// Spinner configuration
	SpinnerIndex = 14
	SpinnerDelay = 100 * time.Millisecond
)

// Color functions using github.com/fatih/color
var (
	// Color functions for different output types
	colorError   = color.New(color.FgRed, color.Bold)
	colorSuccess = color.New(color.FgGreen, color.Bold)
	colorInfo    = color.New(color.FgCyan, color.Bold)
	colorWarning = color.New(color.FgYellow, color.Bold)
	colorCommand = color.New(color.FgCyan)
	colorPrompt  = color.New(color.FgGreen, color.Bold)
	colorHeader  = color.New(color.FgMagenta, color.Bold)
	colorExample = color.New(color.FgBlue)
)

// createSpinner creates and configures a new spinner
func createSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[SpinnerIndex], SpinnerDelay)
	s.Suffix = " " + message
	s.Color("cyan")
	return s
}

// Config holds the application configuration
type Config struct {
	Provider      string `json:"provider"`
	OllamaURL     string `json:"ollama_url"`
	OllamaModel   string `json:"ollama_model"`
	OpenAIKey     string `json:"openai_key"`
	OpenAIModel   string `json:"openai_model"`
	GeminiKey     string `json:"gemini_key"`
	GeminiModel   string `json:"gemini_model"`
	SysPromptFile string `json:"sys_prompt_file"`
}

// LLMClient interface for different LLM providers
type LLMClient interface {
	GenerateCommand(naturalLanguage string) (string, error)
	GetProviderInfo() string
}

// OllamaClient implements LLMClient for Ollama
type OllamaClient struct {
	URL   string
	Model string
}

// OpenAIClient implements LLMClient for OpenAI
type OpenAIClient struct {
	APIKey string
	Model  string
}

// GeminiClient implements LLMClient for Google Gemini
type GeminiClient struct {
	APIKey string
	Model  string
}

// LoadConfig loads configuration from .uc.json file
func LoadConfig(configPath string) (*Config, error) {
	var configFile string

	if configPath != "" {
		// Use explicitly provided config path
		configFile = configPath
	} else {
		// Default to home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("error getting home directory: %v", err)
		}
		configFile = filepath.Join(homeDir, DefaultConfigFile)
	}

	// Check if config file exists, create default if not
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := createDefaultConfig(configFile); err != nil {
			return nil, fmt.Errorf("error creating default config: %v", err)
		}
		fmt.Printf("Created default configuration file: %s\n", configFile)
	}

	// Read and parse JSON config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %v", configFile, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file %s: %v", configFile, err)
	}

	return &config, nil
}

// createDefaultConfig creates a default .uc.json configuration file
func createDefaultConfig(configFile string) error {
	// Get home directory for system prompt file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %v", err)
	}
	sysPromptFile := filepath.Join(homeDir, DefaultPromptFile)

	defaultConfig := Config{
		Provider:      DefaultProvider,
		OllamaURL:     DefaultOllamaURL,
		OllamaModel:   DefaultOllamaModel,
		OpenAIKey:     "",
		OpenAIModel:   DefaultOpenAIModel,
		GeminiKey:     "",
		GeminiModel:   DefaultGeminiModel,
		SysPromptFile: sysPromptFile,
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configFile, data, 0644)
}

// handleSysPromptFile checks if the system prompt file exists, creates it if not, and returns additional prompts
func handleSysPromptFile(filename string) string {
	if filename == "" {
		return ""
	}

	// Expand ~ to home directory if needed
	if strings.HasPrefix(filename, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			colorWarning.Fprintf(os.Stderr, "Warning: Could not get home directory: %v\n", err)
			return ""
		}
		filename = filepath.Join(homeDir, filename[2:])
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// File doesn't exist, create it with default content
		defaultContent := `# UC System Prompts
# Add additional instructions for the LLM here.
# These will be included in all prompts sent to the language model.
# Examples:
# - Be more verbose in explanations
# - Use ffmpeg to process video files
# - Use psql to run SQL queries and manage PostgreSQL databases
# - Add safety warnings for dangerous commands

# Your custom instructions go below:

`
		err := os.WriteFile(filename, []byte(defaultContent), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mWarning: Could not create system prompt file %s: %v\033[0m\n", filename, err)
			return ""
		}
		fmt.Printf("Created system prompt file: %s\n", filename)
		return ""
	}

	// File exists, read it
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mWarning: Could not read system prompt file %s: %v\033[0m\n", filename, err)
		return ""
	}

	// Filter out comments and empty lines
	lines := strings.Split(string(content), "\n")
	var validLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			validLines = append(validLines, line)
		}
	}

	if len(validLines) == 0 {
		return ""
	}

	return strings.Join(validLines, " ")
}

// printError prints an error message in red color
func printError(format string, args ...interface{}) {
	colorError.Fprintf(os.Stderr, format+"\n", args...)
}

// handleCommandError handles command generation and execution errors consistently
func handleCommandError(err error, context string) {
	if context == "Error generating command" {
		printError("%s: %v", context, err)
		fmt.Println("The command can't be executed - failed to generate Unix equivalent.")
	} else if context == "Error executing command" {
		// For execution errors, we've already shown the specific error in ExecuteCommand
		// Just provide a brief summary
		colorWarning.Println("Command execution failed. See error details above.")
	} else {
		printError("%s: %v", context, err)
	}
}

// cleanLLMResponse removes backticks from the start and end of LLM responses
func cleanLLMResponse(response string) string {
	response = strings.TrimSpace(response)

	// Remove backticks from start and end (common in markdown code blocks)
	if strings.HasPrefix(response, "`") && strings.HasSuffix(response, "`") {
		response = strings.TrimPrefix(response, "`")
		response = strings.TrimSuffix(response, "`")
		response = strings.TrimSpace(response)
	}

	// Handle triple backticks (markdown code blocks)
	if strings.HasPrefix(response, "```") && strings.HasSuffix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)

		// Remove language identifier if present (e.g., ```bash)
		lines := strings.Split(response, "\n")
		if len(lines) > 0 {
			// Check if first line is a language identifier
			firstLine := strings.TrimSpace(lines[0])
			if firstLine == "bash" || firstLine == "sh" || firstLine == "shell" {
				lines = lines[1:]
				response = strings.Join(lines, "\n")
				response = strings.TrimSpace(response)
			}
		}
	}

	return response
}

// generatePrompt creates a standardized prompt for all LLM providers
func generatePrompt(naturalLanguage string) string {
	osInfo := detectOS()

	// Get system prompts from file
	config, _ := LoadConfig("")
	additionalPrompts := handleSysPromptFile(config.SysPromptFile)

	basePrompt := fmt.Sprintf(`You are a Unix command generator for %s. Convert the following natural language request into a Unix command appropriate for this operating system. Only return the command, nothing else.`, osInfo)

	if additionalPrompts != "" {
		return fmt.Sprintf(`%s

Additional instructions: %s

Operating System: %s
Natural language request: %s

Unix command:`, basePrompt, additionalPrompts, osInfo, naturalLanguage)
	}
	return fmt.Sprintf(`%s

Operating System: %s
Natural language request: %s

Unix command:`, basePrompt, osInfo, naturalLanguage)
}

// detectOS detects the operating system type and version
func detectOS() string {
	osInfo := runtime.GOOS

	// Try to get more detailed OS information
	switch osInfo {
	case "darwin":
		// macOS
		if version, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			return fmt.Sprintf("macOS %s", strings.TrimSpace(string(version)))
		}
		return "macOS"
	case "linux":
		// Try to detect Linux distribution
		if content, err := exec.Command("cat", "/etc/os-release").Output(); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					name := strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
					return name
				}
			}
		}
		// Fallback to uname
		if uname, err := exec.Command("uname", "-sr").Output(); err == nil {
			return strings.TrimSpace(string(uname))
		}
		return "Linux"
	case "freebsd":
		if uname, err := exec.Command("uname", "-sr").Output(); err == nil {
			return strings.TrimSpace(string(uname))
		}
		return "FreeBSD"
	case "openbsd":
		if uname, err := exec.Command("uname", "-sr").Output(); err == nil {
			return strings.TrimSpace(string(uname))
		}
		return "OpenBSD"
	case "netbsd":
		if uname, err := exec.Command("uname", "-sr").Output(); err == nil {
			return strings.TrimSpace(string(uname))
		}
		return "NetBSD"
	default:
		return osInfo
	}
}

// CreateLLMClient creates the appropriate LLM client based on configuration
func CreateLLMClient(config *Config) (LLMClient, error) {
	switch strings.ToLower(config.Provider) {
	case "ollama":
		return &OllamaClient{URL: config.OllamaURL, Model: config.OllamaModel}, nil
	case "openai":
		if config.OpenAIKey == "" {
			return nil, fmt.Errorf("no OpenAI API key")
		}
		return &OpenAIClient{APIKey: config.OpenAIKey, Model: config.OpenAIModel}, nil
	case "gemini":
		if config.GeminiKey == "" {
			return nil, fmt.Errorf("no Gemini API key")
		}
		return &GeminiClient{APIKey: config.GeminiKey, Model: config.GeminiModel}, nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
}

// GenerateCommand implements LLMClient for Ollama
func (c *OllamaClient) GenerateCommand(naturalLanguage string) (string, error) {
	prompt := generatePrompt(naturalLanguage)

	requestBody := map[string]interface{}{
		"model":  c.Model,
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(c.URL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if responseText, ok := response["response"].(string); ok {
		return cleanLLMResponse(responseText), nil
	}

	return "", fmt.Errorf("unexpected response format from Ollama")
}

// GetProviderInfo returns provider and model information for Ollama
func (c *OllamaClient) GetProviderInfo() string {
	return fmt.Sprintf("Ollama (%s)", c.Model)
}

// GenerateCommand implements LLMClient for OpenAI
func (c *OpenAIClient) GenerateCommand(naturalLanguage string) (string, error) {
	prompt := generatePrompt(naturalLanguage)

	requestBody := map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 100,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	choices, ok := response["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("unexpected response format from OpenAI")
	}

	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content := message["content"].(string)

	return cleanLLMResponse(content), nil
}

// GetProviderInfo returns provider and model information for OpenAI
func (c *OpenAIClient) GetProviderInfo() string {
	return fmt.Sprintf("OpenAI (%s)", c.Model)
}

// GenerateCommand implements LLMClient for Gemini
func (c *GeminiClient) GenerateCommand(naturalLanguage string) (string, error) {
	prompt := generatePrompt(naturalLanguage)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.Model, c.APIKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	candidates, ok := response["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("unexpected response format from Gemini")
	}

	candidate := candidates[0].(map[string]interface{})
	content := candidate["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	part := parts[0].(map[string]interface{})
	text := part["text"].(string)

	return cleanLLMResponse(text), nil
}

// GetProviderInfo returns provider and model information for Gemini
func (c *GeminiClient) GetProviderInfo() string {
	return fmt.Sprintf("Gemini (%s)", c.Model)
}

// ExecuteCommand executes a Unix command
func ExecuteCommand(command string) error {
	if command == "" {
		return fmt.Errorf("no command generated")
	}

	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command")
	}

	// Execute the command through shell to handle wildcards and other shell features
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	// Capture stderr to show it in case of error
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	colorCommand.Printf("%s\n", command)

	// Ensure output is flushed before command execution
	os.Stdout.Sync()

	err := cmd.Run()

	// Ensure output is flushed after command execution
	os.Stdout.Sync()
	os.Stderr.Sync()

	if err != nil {
		// Always display stderr output if there is any
		if stderrOutput := strings.TrimSpace(stderrBuf.String()); stderrOutput != "" {
			// Print to stderr with color and ensure it's flushed
			colorError.Fprintf(os.Stderr, "Error: %s\n", stderrOutput)
			os.Stderr.Sync()
		} else {
			// If no stderr output, show the error from cmd.Run()
			colorError.Fprintf(os.Stderr, "Command failed: %v\n", err)
			os.Stderr.Sync()
		}
		return err
	}

	return nil
}

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "Path to configuration file (default: ~/.uc.json)")
	dryRun := flag.Bool("n", false, "Dry run: show generated command without executing it")
	flag.Parse()

	// Load configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		printError("Error loading configuration: %v", err)
		fmt.Println("Make sure you have a valid .uc.json configuration file.")
		os.Exit(1)
	}

	// Create LLM client
	llmClient, err := CreateLLMClient(config)
	if err != nil {
		printError("Error creating LLM client: %v", err)
		os.Exit(1)
	}

	// Check if we have command line arguments (non-interactive mode)
	args := flag.Args()
	if len(args) >= 1 {
		// Non-interactive mode: execute single command
		naturalLanguage := strings.Join(args, " ")
		fmt.Printf("%s\n", naturalLanguage)
		processCommand(llmClient, naturalLanguage, *dryRun)
		return
	}

	// Interactive mode
	runInteractiveMode(llmClient, *dryRun)
}

// runInteractiveMode runs the interactive REPL loop
func runInteractiveMode(llmClient LLMClient, dryRun bool) {
	osInfo := detectOS()
	llmInfo := llmClient.GetProviderInfo()

	// Colorful startup banner
	fmt.Print("uc ")
	fmt.Printf("(%s)", osInfo)
	fmt.Print(" - ")
	fmt.Printf("%s\n", llmInfo)
	fmt.Println("Type your natural language commands below. Type 'exit' to exit, 'help' for help.")

	// Get user home directory for history file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		printError("Error getting home directory: %v", err)
		return
	}
	historyFile := filepath.Join(homeDir, DefaultHistoryFile)

	// Helper function to get prompt with appropriate color
	getPrompt := func(dryRun bool) string {
		if dryRun {
			return colorPrompt.Sprint("uc> ")
		}
		return NormalPrompt
	}

	// Configure readline
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          getPrompt(dryRun),
		HistoryFile:     historyFile,
		AutoComplete:    nil,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		printError("Error initializing readline: %v", err)
		return
	}
	defer rl.Close()

	for {
		// Read input from user with readline (supports history and arrow keys)
		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(input) == 0 {
					fmt.Println("\nGoodbye!")
					break
				} else {
					continue
				}
			} else if err == io.EOF {
				fmt.Println("\nGoodbye!")
				break
			}
			printError("Error reading input: %v", err)
			continue
		}

		// Trim whitespace
		input = strings.TrimSpace(input)

		// Check for exit commands
		if input == "" {
			continue
		}

		if strings.ToLower(input) == CmdExit {
			fmt.Println("Goodbye!")
			break
		}

		if strings.ToLower(input) == CmdHelp {
			showHelp()
			continue
		}

		if strings.ToLower(input) == CmdDryRun {
			dryRun = !dryRun
			// Update the prompt color based on dry-run mode
			rl.SetPrompt(getPrompt(dryRun))
			if dryRun {
				colorWarning.Println("Dry-run mode enabled. Commands will be shown but not executed.")
			} else {
				colorSuccess.Println("Dry-run mode disabled. Commands will be executed normally.")
			}
			continue
		}

		// Process the command
		processCommand(llmClient, input, dryRun)
		fmt.Println() // Add blank line for readability
	}
}

// processCommand processes a single natural language command
func processCommand(llmClient LLMClient, naturalLanguage string, dryRun bool) {
	// Create and start spinner while generating command
	s := createSpinner("Generating command...")
	s.Start()

	// Generate Unix command using LLM
	unixCommand, err := llmClient.GenerateCommand(naturalLanguage)

	// Stop spinner
	s.Stop()

	if err != nil {
		handleCommandError(err, "Error generating command")
		return
	}

	if strings.TrimSpace(unixCommand) == "" {
		printError("LLM returned empty command")
		return
	}

	if dryRun {
		// Dry run: just show the command without executing
		colorWarning.Print("[dry run] ")
		colorCommand.Printf("%s\n", unixCommand)
		return
	}

	// Execute the generated command
	if err := ExecuteCommand(unixCommand); err != nil {
		handleCommandError(err, "Error executing command")
	}
}

// showHelp displays help information
func showHelp() {
	colorHeader.Println("Help - Unix Commands in Natural Language")
	fmt.Println()

	// Interactive commands
	colorInfo.Println("Interactive Commands:")
	colorSuccess.Printf("  %-12s", CmdHelp)
	fmt.Println(" - Show this help message")
	colorSuccess.Printf("  %-12s", CmdDryRun)
	fmt.Println(" - Toggle dry-run mode (show commands without executing)")
	colorSuccess.Printf("  %-12s", CmdExit)
	fmt.Println(" - Exit the program")
	fmt.Println()

	// Example commands
	colorInfo.Println("Example Natural Language Commands:")
	examples := []string{
		"list all files in the current directory",
		"show me the contents of README.md",
		"find all Go files in this directory",
		"count the number of files in this directory",
		"show running processes",
		"check disk usage",
		"what's the current date and time",
		"ping google.com",
	}

	for _, example := range examples {
		colorExample.Printf("  - %s\n", example)
	}
	fmt.Println()
}
