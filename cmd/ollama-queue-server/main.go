package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
	"github.com/liliang-cn/ollama-queue/pkg/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed static
var staticFiles embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	cfgFile    string
	configDir  string
	dataDir    string
	ollamaHost string
	verbose    bool
	port       string
	host       string
)

var manager *queue.QueueManager

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ollama-queue-server",
	Short: "Ollama Queue Server - Task queue management server for Ollama models",
	Long: `Ollama Queue Server is a high-performance task queue management server designed for Ollama models.
It provides HTTP API endpoints for task submission, monitoring, and management with built-in web UI.

The server handles:
- Task queuing and prioritization
- Cron-based recurring tasks  
- Remote task scheduling
- Real-time task monitoring
- Web-based management interface`,
	RunE: runServer,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ollama-queue/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "config directory (default is $HOME/.ollama-queue)")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "", "data directory (default is ./data)")
	rootCmd.PersistentFlags().StringVar(&ollamaHost, "ollama-host", "http://localhost:11434", "Ollama host URL")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	
	// Server-specific flags
	rootCmd.Flags().StringVar(&port, "port", "8080", "Server port")
	rootCmd.Flags().StringVar(&host, "host", "localhost", "Server host")

	// Bind flags to viper
	viper.BindPFlag("ollama.host", rootCmd.PersistentFlags().Lookup("ollama-host"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("server.port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("server.host", rootCmd.Flags().Lookup("host"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configPath := home + "/.ollama-queue"
		if configDir != "" {
			configPath = configDir
		}

		viper.AddConfigPath(configPath)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("OLLAMA_QUEUE")

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			log.Printf("Using config file: %s", viper.ConfigFileUsed())
		}
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	log.Printf("Starting Ollama Queue Server...")
	
	// Initialize configuration
	config := models.DefaultConfig()
	if dataDir != "" {
		config.StoragePath = dataDir
	}
	if ollamaHost != "" {
		config.OllamaHost = ollamaHost
	}
	
	// Set server address
	config.ListenAddr = host + ":" + port

	// Create queue manager
	var err error
	manager, err = queue.NewQueueManager(config)
	if err != nil {
		return err
	}
	defer manager.Close()

	// Start queue manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := manager.Start(ctx); err != nil {
		return err
	}

	// Create broadcaster for WebSocket
	broadcaster, err := ui.NewBroadcaster(manager)
	if err != nil {
		return err
	}
	defer broadcaster.Stop()

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	
	// Setup HTML template
	templ := template.Must(template.New("").ParseFS(staticFiles, "static/*.html"))
	router.SetHTMLTemplate(templ)

	// Setup routes
	setupRoutes(router, broadcaster)

	// Start HTTP server
	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: router,
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on http://%s", config.ListenAddr)
		log.Printf("Web UI available at http://%s", config.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
	return nil
}

func setupRoutes(router *gin.Engine, broadcaster *ui.Broadcaster) {
	// Web UI
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer ws.Close()

		broadcaster.AddClient(ws)
		defer broadcaster.RemoveClient(ws)

		// Keep the connection alive
		for {
			if _, _, err := ws.NextReader(); err != nil {
				break
			}
		}
	})

	// API routes - basic task management for now
	api := router.Group("/api")
	{
		api.POST("/tasks/:id/cancel", func(c *gin.Context) {
			taskID := c.Param("id")
			if err := manager.CancelTask(taskID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		api.GET("/tasks", func(c *gin.Context) {
			filter := models.TaskFilter{}
			tasks, err := manager.ListTasks(filter)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, tasks)
		})

		api.GET("/stats", func(c *gin.Context) {
			stats, err := manager.GetQueueStats()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, stats)
		})
	}
}