package main

import (
	"fmt"
	"log"

	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/executor"
)

func main() {
	fmt.Println("=== Ollama Queue 作为通用任务队列库使用示例 ===")
	
	// 1. 创建执行器注册表
	registry := executor.NewExecutorRegistry()
	
	// 2. 创建和注册执行器插件
	fmt.Println("📋 注册执行器插件...")
	
	// 注册脚本执行器（最简单的，不依赖外部服务）
	scriptConfig := executor.ScriptConfig{
		WorkingDir: "./",
		AllowedExts: []string{".sh", ".py", ".js"},
	}
	
	scriptPlugin, err := executor.NewScriptPlugin(scriptConfig)
	if err != nil {
		log.Printf("创建脚本执行器失败: %v", err)
	} else {
		if err := registry.Register(models.ExecutorTypeScript, scriptPlugin); err != nil {
			log.Printf("注册脚本执行器失败: %v", err)
		} else {
			fmt.Printf("✅ 脚本执行器注册成功\n")
		}
	}
	
	// 3. 列出所有可用执行器
	executors := registry.ListExecutors()
	fmt.Printf("📊 可用执行器: %d个\n", len(executors))
	for _, exec := range executors {
		fmt.Printf("  - 类型: %s\n", exec.Type)
		fmt.Printf("    名称: %s\n", exec.Name)
		fmt.Printf("    动作: %v\n", exec.Actions)
		fmt.Printf("    描述: %s\n\n", exec.Description)
	}
	
	// 4. 创建通用任务
	fmt.Println("🚀 创建通用任务...")
	
	// 创建脚本执行任务
	scriptTask := models.CreateGenericTask(
		models.ExecutorTypeScript,
		models.ActionExecute,
		map[string]interface{}{
			"command": "echo 'Hello from Universal Task Queue!'",
			"env": map[string]interface{}{
				"MESSAGE": "This is from environment variable",
			},
		},
	)
	scriptTask.Priority = models.PriorityHigh
	
	fmt.Printf("✅ 创建任务: %s\n", scriptTask.ID)
	fmt.Printf("  执行器: %s\n", scriptTask.ExecutorType)
	fmt.Printf("  动作: %s\n", scriptTask.Action)
	fmt.Printf("  优先级: %d\n", scriptTask.Priority)
	
	// 5. 验证任务可以被执行
	if registry.CanExecuteTask(scriptTask) {
		fmt.Printf("✅ 任务可以被执行\n")
		
		// 获取执行器并验证载荷
		executor, exists := registry.GetExecutor(scriptTask.ExecutorType)
		if exists {
			if err := executor.Validate(scriptTask.Action, scriptTask.Payload); err != nil {
				log.Printf("任务载荷验证失败: %v", err)
			} else {
				fmt.Printf("✅ 任务载荷验证通过\n")
			}
		}
	} else {
		fmt.Printf("❌ 任务无法执行\n")
	}
	
	// 6. 展示任务适配器功能
	fmt.Println("\n🔄 测试向后兼容性...")
	
	adapter := models.NewTaskAdapter()
	
	// 创建传统任务
	legacyTask := &models.Task{
		ID:       models.GenerateID(),
		Type:     models.TaskTypeGenerate,
		Model:    "qwen3",
		Priority: models.PriorityNormal,
		Payload: map[string]interface{}{
			"prompt": "Write a hello world program",
			"system": "You are a coding assistant",
		},
	}
	
	// 转换为通用任务
	genericTask, err := adapter.ConvertToGeneric(legacyTask)
	if err != nil {
		log.Printf("任务转换失败: %v", err)
	} else {
		fmt.Printf("✅ 传统任务成功转换为通用任务\n")
		fmt.Printf("  %s -> %s:%s\n", legacyTask.Type, genericTask.ExecutorType, genericTask.Action)
	}
	
	// 转换回传统任务
	convertedBack, err := adapter.ConvertToLegacy(genericTask)
	if err != nil {
		log.Printf("回转换失败: %v", err)
	} else if convertedBack.Type == legacyTask.Type {
		fmt.Printf("✅ 双向转换成功\n")
	}
	
	fmt.Println("\n🎉 ollama-queue 现在是通用任务队列库！")
	fmt.Println("📚 支持的执行器类型:")
	fmt.Println("  - ollama: 本地AI模型 (chat, generate, embed)")
	fmt.Println("  - openai: OpenAI API (chat, generate)")
	fmt.Println("  - script: 脚本执行 (execute)")
	fmt.Println("  - 更多: 可扩展添加任意执行器类型")
	
	fmt.Println("\n💡 使用方式:")
	fmt.Println("  1. 作为库: 导入pkg包，创建执行器注册表")
	fmt.Println("  2. 作为服务: 运行ollama-queue-server")
	fmt.Println("  3. 作为客户端: 使用ollama-queue CLI")
	fmt.Println("  4. 向后兼容: 现有代码无需修改")
}