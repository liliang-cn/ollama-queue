
package cmd

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
	"github.com/liliang-cn/ollama-queue/pkg/ui"
	"github.com/spf13/cobra"
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

var manager *queue.QueueManager

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts a web server to view the queue",
	Run: func(cmd *cobra.Command, args []string) {
		config := models.DefaultConfig()
		if dataDir != "" {
			config.StoragePath = dataDir
		}
		if ollamaHost != "" {
			config.OllamaHost = ollamaHost
		}

		var err error
		manager, err = queue.NewQueueManager(config)
		if err != nil {
			log.Fatalf("Failed to create new manager: %v", err)
		}
		defer manager.Close()

		if err := manager.Start(context.Background()); err != nil {
			log.Fatalf("Failed to start manager: %v", err)
		}

		broadcaster, err := ui.NewBroadcaster(manager)
		if err != nil {
			log.Fatalf("Failed to create broadcaster: %v", err)
		}
		defer 		broadcaster.Stop()

		gin.SetMode(gin.ReleaseMode)
		router := gin.Default()
		templ := template.Must(template.New("").ParseFS(staticFiles, "static/*.html"))
		router.SetHTMLTemplate(templ)

		router.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", nil)
		})

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
				// Read messages from the client (and ignore them)
				// This is to detect when the client closes the connection
				if _, _, err := ws.NextReader(); err != nil {
					break
				}
			}
		})

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

			api.POST("/tasks/:id/priority", func(c *gin.Context) {
				taskID := c.Param("id")
				var req struct {
					Priority models.Priority `json:"priority"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if err := manager.UpdateTaskPriority(taskID, req.Priority); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			api.POST("/tasks", func(c *gin.Context) {
				var task models.Task
				if err := c.ShouldBindJSON(&task); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if _, err := manager.SubmitTask(&task); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			api.GET("/tasks", func(c *gin.Context) {
				tasks, err := manager.ListTasks(models.TaskFilter{})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, tasks)
			})

			api.GET("/tasks/:id", func(c *gin.Context) {
				taskID := c.Param("id")
				task, err := manager.GetTask(taskID)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, task)
			})

			api.GET("/status", func(c *gin.Context) {
				stats, err := manager.GetQueueStats()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, stats)
			})
		}

		port, _ := cmd.Flags().GetString("port")
		log.Printf("Server started on :%s", port)
		if err := router.Run(":" + port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("port", "8080", "Port to run the web server on")
}
