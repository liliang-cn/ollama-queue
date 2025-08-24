package executor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	
	"github.com/liliang-cn/ollama-queue/pkg/models"
)

// ScriptPlugin implements the GenericExecutor interface for script execution
type ScriptPlugin struct {
	metadata    *ExecutorMetadata
	workingDir  string
	timeout     time.Duration
	allowedExts map[string]bool
}

// ScriptConfig holds script executor configuration
type ScriptConfig struct {
	WorkingDir   string        `json:"working_dir"`   // Directory where scripts are located
	Timeout      time.Duration `json:"timeout"`       // Maximum execution time
	AllowedExts  []string      `json:"allowed_exts"`  // Allowed file extensions (.py, .sh, .js, etc.)
}

// NewScriptPlugin creates a new script executor plugin
func NewScriptPlugin(config ScriptConfig) (*ScriptPlugin, error) {
	if config.WorkingDir == "" {
		config.WorkingDir = "./scripts"
	}
	
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	
	// Default allowed extensions
	if len(config.AllowedExts) == 0 {
		config.AllowedExts = []string{".py", ".sh", ".js", ".rb", ".go"}
	}
	
	// Create allowed extensions map
	allowedExts := make(map[string]bool)
	for _, ext := range config.AllowedExts {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		allowedExts[ext] = true
	}
	
	// Ensure working directory exists
	if err := os.MkdirAll(config.WorkingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}
	
	plugin := &ScriptPlugin{
		workingDir:  config.WorkingDir,
		timeout:     config.Timeout,
		allowedExts: allowedExts,
		metadata: &ExecutorMetadata{
			Name:        "Script Executor",
			Type:        models.ExecutorTypeScript,
			Description: "Executes scripts and commands",
			Version:     "1.0.0",
			Actions: []ExecutorAction{
				models.ActionExecute,
			},
			Config: map[string]any{
				"working_dir":   config.WorkingDir,
				"timeout":       config.Timeout.String(),
				"allowed_exts":  config.AllowedExts,
			},
		},
	}
	
	return plugin, nil
}

// GetMetadata returns information about this executor
func (p *ScriptPlugin) GetMetadata() *ExecutorMetadata {
	return p.metadata
}

// CanExecute checks if the executor can handle a specific action
func (p *ScriptPlugin) CanExecute(action ExecutorAction) bool {
	return action == models.ActionExecute
}

// Execute executes a script task synchronously
func (p *ScriptPlugin) Execute(ctx context.Context, task *GenericTask) (*TaskResult, error) {
	result := &TaskResult{
		TaskID:  task.ID,
		Success: false,
	}
	
	if task.Action != models.ActionExecute {
		err := fmt.Errorf("unsupported action: %s", task.Action)
		result.Error = err.Error()
		return result, err
	}
	
	output, err := p.executeScript(ctx, task)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	
	result.Data = map[string]interface{}{
		"output": output,
		"status": "completed",
	}
	result.Success = true
	
	return result, nil
}

// ExecuteStream executes a script with streaming output
func (p *ScriptPlugin) ExecuteStream(ctx context.Context, task *GenericTask) (<-chan *StreamChunk, error) {
	outputChan := make(chan *StreamChunk, 10)
	
	go func() {
		defer close(outputChan)
		
		if task.Action != models.ActionExecute {
			outputChan <- &StreamChunk{
				Error: fmt.Errorf("unsupported action: %s", task.Action),
				Done:  true,
			}
			return
		}
		
		p.executeScriptStream(ctx, task, outputChan)
	}()
	
	return outputChan, nil
}

// Validate validates that a task payload is correct for this executor
func (p *ScriptPlugin) Validate(action ExecutorAction, payload map[string]any) error {
	if action != models.ActionExecute {
		return fmt.Errorf("unsupported action: %s", action)
	}
	
	// Check for script or command
	if _, hasScript := payload["script"]; hasScript {
		return p.validateScriptPayload(payload)
	}
	
	if _, hasCommand := payload["command"]; hasCommand {
		return p.validateCommandPayload(payload)
	}
	
	return fmt.Errorf("missing script or command in payload")
}

// Close cleans up executor resources
func (p *ScriptPlugin) Close() error {
	// No cleanup needed for script executor
	return nil
}

// Private execution methods
func (p *ScriptPlugin) executeScript(ctx context.Context, task *GenericTask) (string, error) {
	// Determine execution type
	if script, ok := task.Payload["script"].(string); ok {
		return p.runScriptFile(ctx, script, task.Payload)
	}
	
	if command, ok := task.Payload["command"].(string); ok {
		return p.runCommand(ctx, command, task.Payload)
	}
	
	return "", fmt.Errorf("no valid script or command found in payload")
}

func (p *ScriptPlugin) executeScriptStream(ctx context.Context, task *GenericTask, outputChan chan<- *StreamChunk) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	
	var cmd *exec.Cmd
	
	if script, ok := task.Payload["script"].(string); ok {
		var err error
		cmd, err = p.prepareScriptCommand(timeoutCtx, script, task.Payload)
		if err != nil {
			outputChan <- &StreamChunk{Error: err, Done: true}
			return
		}
	} else if command, ok := task.Payload["command"].(string); ok {
		cmd = p.prepareCommand(timeoutCtx, command, task.Payload)
	} else {
		outputChan <- &StreamChunk{
			Error: fmt.Errorf("no valid script or command found"),
			Done:  true,
		}
		return
	}
	
	// Set up pipes for streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		outputChan <- &StreamChunk{Error: err, Done: true}
		return
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		outputChan <- &StreamChunk{Error: err, Done: true}
		return
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		outputChan <- &StreamChunk{Error: err, Done: true}
		return
	}
	
	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case outputChan <- &StreamChunk{Data: line + "\n"}:
			case <-timeoutCtx.Done():
				return
			}
		}
	}()
	
	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case outputChan <- &StreamChunk{Data: "[STDERR] " + line + "\n"}:
			case <-timeoutCtx.Done():
				return
			}
		}
	}()
	
	// Wait for completion
	err = cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				outputChan <- &StreamChunk{
					Error: fmt.Errorf("command exited with code %d", status.ExitStatus()),
					Done:  true,
				}
				return
			}
		}
		outputChan <- &StreamChunk{Error: err, Done: true}
		return
	}
	
	outputChan <- &StreamChunk{Done: true}
}

func (p *ScriptPlugin) runScriptFile(ctx context.Context, scriptPath string, payload map[string]any) (string, error) {
	// Validate script path
	if !p.isValidScriptPath(scriptPath) {
		return "", fmt.Errorf("invalid or unsafe script path: %s", scriptPath)
	}
	
	fullPath := filepath.Join(p.workingDir, scriptPath)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("script file not found: %s", scriptPath)
	}
	
	// Check extension
	ext := filepath.Ext(scriptPath)
	if !p.allowedExts[ext] {
		return "", fmt.Errorf("script extension %s not allowed", ext)
	}
	
	cmd, err := p.prepareScriptCommand(ctx, scriptPath, payload)
	if err != nil {
		return "", err
	}
	
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (p *ScriptPlugin) runCommand(ctx context.Context, command string, payload map[string]any) (string, error) {
	cmd := p.prepareCommand(ctx, command, payload)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (p *ScriptPlugin) prepareScriptCommand(ctx context.Context, scriptPath string, payload map[string]any) (*exec.Cmd, error) {
	fullPath := filepath.Join(p.workingDir, scriptPath)
	ext := filepath.Ext(scriptPath)
	
	var cmd *exec.Cmd
	
	// Determine interpreter based on extension
	switch ext {
	case ".py":
		cmd = exec.CommandContext(ctx, "python3", fullPath)
	case ".sh":
		cmd = exec.CommandContext(ctx, "bash", fullPath)
	case ".js":
		cmd = exec.CommandContext(ctx, "node", fullPath)
	case ".rb":
		cmd = exec.CommandContext(ctx, "ruby", fullPath)
	case ".go":
		cmd = exec.CommandContext(ctx, "go", "run", fullPath)
	default:
		return nil, fmt.Errorf("unsupported script type: %s", ext)
	}
	
	// Set working directory
	cmd.Dir = p.workingDir
	
	// Add environment variables from payload
	if env, ok := payload["env"].(map[string]interface{}); ok {
		cmd.Env = os.Environ()
		for key, value := range env {
			if strValue, ok := value.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, strValue))
			}
		}
	}
	
	// Add arguments from payload
	if args, ok := payload["args"].([]interface{}); ok {
		for _, arg := range args {
			if strArg, ok := arg.(string); ok {
				cmd.Args = append(cmd.Args, strArg)
			}
		}
	}
	
	return cmd, nil
}

func (p *ScriptPlugin) prepareCommand(ctx context.Context, command string, payload map[string]any) *exec.Cmd {
	// Simple command execution using shell
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = p.workingDir
	
	// Add environment variables
	if env, ok := payload["env"].(map[string]interface{}); ok {
		cmd.Env = os.Environ()
		for key, value := range env {
			if strValue, ok := value.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, strValue))
			}
		}
	}
	
	return cmd
}

// Validation methods
func (p *ScriptPlugin) validateScriptPayload(payload map[string]any) error {
	script, ok := payload["script"].(string)
	if !ok {
		return fmt.Errorf("script must be a string")
	}
	
	if !p.isValidScriptPath(script) {
		return fmt.Errorf("invalid script path: %s", script)
	}
	
	ext := filepath.Ext(script)
	if !p.allowedExts[ext] {
		return fmt.Errorf("script extension %s not allowed", ext)
	}
	
	return nil
}

func (p *ScriptPlugin) validateCommandPayload(payload map[string]any) error {
	_, ok := payload["command"].(string)
	if !ok {
		return fmt.Errorf("command must be a string")
	}
	
	return nil
}

func (p *ScriptPlugin) isValidScriptPath(path string) bool {
	// Security check: no path traversal
	if strings.Contains(path, "..") {
		return false
	}
	
	// No absolute paths
	if filepath.IsAbs(path) {
		return false
	}
	
	// Clean the path
	clean := filepath.Clean(path)
	if clean != path {
		return false
	}
	
	return true
}